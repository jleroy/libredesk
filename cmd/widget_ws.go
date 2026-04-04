package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
	"time"

	realip "github.com/ferluci/fast-realip"

	"github.com/abhinavxd/libredesk/internal/httputil"
	"github.com/abhinavxd/libredesk/internal/inbox/channel/livechat"
	"github.com/fasthttp/websocket"
	"github.com/zerodha/fastglue"
)

const (
	WidgetMsgTypeJoin      = "join"
	WidgetMsgTypeTyping    = "typing"
	WidgetMsgTypePing      = "ping"
	WidgetMsgTypePong      = "pong"
	WidgetMsgTypeError     = "error"
	WidgetMsgTypeJoined    = "joined"
	WidgetMsgTypePageVisit = "page_visit"

	pageVisitRedisKeyPrefix = "page_visits:"
	maxPageVisits           = 10
	pageVisitTTL            = 24 * time.Hour
	wsReadDeadline          = 20 * time.Second
)

type WidgetMessage struct {
	Type string          `json:"type"`
	JWT  string          `json:"jwt,omitempty"`
	Data json.RawMessage `json:"data"`
}

type WidgetInboxJoinRequest struct {
	InboxID string `json:"inbox_id"`
}

type WidgetTypingData struct {
	ConversationUUID string `json:"conversation_uuid"`
	IsTyping         bool   `json:"is_typing"`
}

type WidgetPageVisitData struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}

// safeConn wraps a WebSocket connection with a mutex for concurrent-safe writes.
type safeConn struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func (sc *safeConn) WriteJSON(v any) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.conn.WriteJSON(v)
}

func (sc *safeConn) WriteMessage(msgType int, data []byte) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.conn.WriteMessage(msgType, data)
}

func handleWidgetWS(r *fastglue.Request) error {
	var app = r.Context.(*App)

	clientIP := realip.FromRequest(r.RequestCtx)

	if err := upgrader.Upgrade(r.RequestCtx, func(conn *websocket.Conn) {
		sc := &safeConn{conn: conn}

		var (
			client   *livechat.Client
			liveChat *livechat.LiveChat
			inboxUUID string
			userID    int
		)

		defer func() {
			conn.Close()
			if client != nil && liveChat != nil {
				liveChat.RemoveClient(client)
				client.CloseChannel()
			}
		}()

		for {
			conn.SetReadDeadline(time.Now().Add(wsReadDeadline))
			var msg WidgetMessage
			if err := conn.ReadJSON(&msg); err != nil {
				app.lo.Debug("widget websocket connection closed", "error", err)
				break
			}

			switch msg.Type {
			case WidgetMsgTypeJoin:
				// Clean up previous client on re-join.
				if client != nil && liveChat != nil {
					liveChat.RemoveClient(client)
					client.CloseChannel()
					client = nil
					liveChat = nil
				}

				joinedClient, joinedLiveChat, joinedInboxUUID, joinedUserID, err := handleInboxJoin(app, sc, msg.Data, msg.JWT, clientIP)
				if err != nil {
					app.lo.Error("error handling widget join", "error", err)
					sendWidgetError(sc, "Failed to join conversation")
					continue
				}
				client = joinedClient
				liveChat = joinedLiveChat
				inboxUUID = joinedInboxUUID
				userID = joinedUserID

			case WidgetMsgTypeTyping:
				if userID == 0 || inboxUUID == "" {
					continue
				}
				handleWidgetTyping(app, msg.Data, inboxUUID, userID, msg.JWT)

			case WidgetMsgTypePageVisit:
				if userID > 0 {
					handleWidgetPageVisit(app, msg.Data, userID)
				}

			case WidgetMsgTypePing:
				if userID > 0 {
					wasOffline := app.user.IsOffline(userID)
					if err := app.user.UpdateLastActive(userID); err != nil {
						app.lo.Error("error updating user last active timestamp", "user_id", userID, "error", err)
					} else if wasOffline {
						app.conversation.BroadcastContactUpdate(userID, map[string]any{"availability_status": "online"})
					}
				}

				if err := sc.WriteJSON(WidgetMessage{Type: WidgetMsgTypePong}); err != nil {
					app.lo.Error("error writing pong to widget client", "error", err)
				}
			}
		}
	}); err != nil {
		app.lo.Error("error upgrading widget websocket connection", "error", err)
	}
	return nil
}

func handleInboxJoin(app *App, sc *safeConn, data json.RawMessage, jwtToken, clientIP string) (*livechat.Client, *livechat.LiveChat, string, int, error) {
	var joinData WidgetInboxJoinRequest
	if err := json.Unmarshal(data, &joinData); err != nil {
		return nil, nil, "", 0, fmt.Errorf("invalid join data: %w", err)
	}

	inbox, err := app.inbox.GetDBRecord(joinData.InboxID)
	if err != nil {
		return nil, nil, "", 0, fmt.Errorf("inbox not found: %w", err)
	}
	if !inbox.Enabled {
		return nil, nil, "", 0, fmt.Errorf("inbox is not enabled")
	}

	claims, err := verifyStandardJWT(jwtToken, inbox.Secret.String)
	if err != nil {
		return nil, nil, "", 0, fmt.Errorf("JWT validation failed: %w", err)
	}

	user, err := resolveUserFromClaims(app, claims)
	if err != nil {
		return nil, nil, "", 0, fmt.Errorf("failed to resolve user ID from claims: %w", err)
	}

	var config livechat.Config
	if err := json.Unmarshal(inbox.Config, &config); err == nil {
		if len(config.BlockedIPs) > 0 && httputil.IsIPBlocked(clientIP, config.BlockedIPs) {
			return nil, nil, "", 0, fmt.Errorf("IP address is blocked")
		}
	}

	lcInbox, err := app.inbox.Get(inbox.ID)
	if err != nil {
		return nil, nil, "", 0, fmt.Errorf("live chat inbox not found: %w", err)
	}

	liveChat, ok := lcInbox.(*livechat.LiveChat)
	if !ok {
		return nil, nil, "", 0, fmt.Errorf("inbox is not a live chat inbox")
	}

	userIDStr := fmt.Sprintf("%d", user.ID)
	client, err := liveChat.AddClient(userIDStr)
	if err != nil {
		return nil, nil, "", 0, fmt.Errorf("adding client to live chat: %w", err)
	}

	go func() {
		for msgData := range client.Channel {
			if err := sc.WriteMessage(websocket.TextMessage, msgData); err != nil {
				app.lo.Error("error forwarding message to widget client", "error", err)
				return
			}
		}
	}()

	if err := sc.WriteJSON(WidgetMessage{
		Type: WidgetMsgTypeJoined,
		Data: json.RawMessage(`{"message":"namaste!"}`),
	}); err != nil {
		return nil, nil, "", 0, err
	}

	app.lo.Debug("widget client joined live chat", "user_id", userIDStr, "inbox_uuid", joinData.InboxID)

	return client, liveChat, joinData.InboxID, user.ID, nil
}

func handleWidgetTyping(app *App, data json.RawMessage, inboxUUID string, userID int, jwtToken string) {
	var typingData WidgetTypingData
	if err := json.Unmarshal(data, &typingData); err != nil || typingData.ConversationUUID == "" {
		return
	}

	if _, err := validateWidgetMessageJWT(app, jwtToken, inboxUUID); err != nil {
		return
	}

	conversation, err := app.conversation.GetConversation(0, typingData.ConversationUUID, "")
	if err != nil || conversation.ContactID != userID {
		return
	}

	app.conversation.BroadcastTypingToConversation(typingData.ConversationUUID, typingData.IsTyping, false)
}

func validateWidgetMessageJWT(app *App, jwtToken string, inboxUUID string) (Claims, error) {
	if inboxUUID == "" {
		return Claims{}, fmt.Errorf("inbox UUID is required for JWT validation")
	}

	inbox, err := app.inbox.GetDBRecord(inboxUUID)
	if err != nil {
		return Claims{}, fmt.Errorf("inbox not found: %w", err)
	}

	return verifyStandardJWT(jwtToken, inbox.Secret.String)
}

func sendWidgetError(sc *safeConn, message string) {
	data, _ := json.Marshal(map[string]string{"message": message})
	sc.WriteJSON(WidgetMessage{
		Type: WidgetMsgTypeError,
		Data: data,
	})
}

func handleWidgetPageVisit(app *App, data json.RawMessage, contactID int) {
	var visit WidgetPageVisitData
	if err := json.Unmarshal(data, &visit); err != nil || visit.URL == "" {
		return
	}

	if len(visit.URL) > 2048 {
		visit.URL = visit.URL[:2048]
	}
	if len(visit.Title) > 256 {
		visit.Title = visit.Title[:256]
	}

	parsedURL, err := url.Parse(visit.URL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		return
	}

	entry, _ := json.Marshal(map[string]string{
		"url":   visit.URL,
		"title": visit.Title,
		"time":  time.Now().UTC().Format(time.RFC3339),
	})

	redisCtx := context.Background()
	key := fmt.Sprintf("%s%d", pageVisitRedisKeyPrefix, contactID)
	pipe := app.redis.Pipeline()
	pipe.LPush(redisCtx, key, string(entry))
	pipe.LTrim(redisCtx, key, 0, maxPageVisits-1)
	pipe.Expire(redisCtx, key, pageVisitTTL)
	lrangeCmd := pipe.LRange(redisCtx, key, 0, maxPageVisits-1)
	pipe.Exec(redisCtx)

	entries, err := lrangeCmd.Result()
	if err != nil {
		return
	}
	pages := make([]map[string]string, 0, len(entries))
	for _, e := range entries {
		var p map[string]string
		if err := json.Unmarshal([]byte(e), &p); err == nil {
			pages = append(pages, p)
		}
	}
	app.conversation.BroadcastContactUpdate(contactID, map[string]any{"page_visits": pages})
}

func getPageVisitsFromRedis(app *App, contactID int) []map[string]string {
	redisCtx := context.Background()
	key := fmt.Sprintf("%s%d", pageVisitRedisKeyPrefix, contactID)
	entries, err := app.redis.LRange(redisCtx, key, 0, maxPageVisits-1).Result()
	if err != nil {
		return nil
	}
	pages := make([]map[string]string, 0, len(entries))
	for _, e := range entries {
		var p map[string]string
		if err := json.Unmarshal([]byte(e), &p); err == nil {
			pages = append(pages, p)
		}
	}
	return pages
}
