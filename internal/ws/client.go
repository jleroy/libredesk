package ws

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/abhinavxd/libredesk/internal/ws/models"
	"github.com/fasthttp/websocket"
)

const (
	pongWait   = 60 * time.Second
	pingPeriod = 25 * time.Second
	writeWait  = 10 * time.Second
	// 64 KiB
	maxMessageSize  = 64 << 10
	maxListSubUUIDs = 500
)

// SafeBool is a thread-safe boolean.
type SafeBool struct {
	flag bool
	mu   sync.RWMutex
}

// Set sets the value of the SafeBool.
func (b *SafeBool) Set(value bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.flag = value
}

// Get returns the value of the SafeBool.
func (b *SafeBool) Get() bool {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.flag
}

// Client is a single connected WS user.
type Client struct {
	// Client ID.
	ID int

	// Hub.
	Hub *Hub

	// WebSocket connection.
	Conn *websocket.Conn

	// To prevent pushes to the channel.
	Closed SafeBool

	// Buffered channel of outbound ws messages.
	Send chan models.WSMessage
}

// Serve handles heartbeats and sending messages to the client.
func (c *Client) Serve() {
	// Write deadlines below: a peer that stops reading would otherwise block this goroutine and hold the socket open indefinitely.
	var heartBeatTicker = time.NewTicker(pingPeriod)
	defer heartBeatTicker.Stop()
	defer c.Conn.Close()

	for {
		select {
		case <-heartBeatTicker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		case msg, ok := <-c.Send:
			if !ok {
				return
			}
			c.Conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.Conn.WriteMessage(msg.MessageType, msg.Data); err != nil {
				return
			}
		}
	}
}

// Listen is a block method that listens for incoming messages from the client.
func (c *Client) Listen() {
	// A peer that disappears without closing (slept laptop, dropped wifi) leaves this read blocked forever unless a pong refreshes the deadline.
	c.Conn.SetReadLimit(maxMessageSize)
	if err := c.Conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		c.Hub.RemoveClient(c)
		c.close()
		return
	}
	c.Conn.SetPongHandler(func(string) error {
		return c.Conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		msgType, msg, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}

		if msgType == websocket.TextMessage {
			c.processIncomingMessage(msg)
		} else {
			c.Hub.RemoveClient(c)
			c.close()
			return
		}
	}
	c.Hub.RemoveClient(c)
	c.close()
}

// processIncomingMessage processes incoming messages from the client.
func (c *Client) processIncomingMessage(data []byte) {
	if string(data) == "ping" {
		if _, err := c.Hub.userStore.UpdateLastActive(c.ID); err != nil {
			c.Hub.lo.Error("UpdateLastActive failed", "client_id", c.ID, "error", err)
		}
		c.SendMessage([]byte("pong"), websocket.TextMessage)
		return
	}

	// Try to parse as JSON message
	var msg models.IncomingMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		c.SendError("invalid message format")
		return
	}

	switch msg.Type {
	case models.MessageTypeConversationSubscribe:
		c.handleConversationSubscribe(msg.Data)
	case models.MessageTypeListSubscribeReplace:
		c.handleListSubscribe(msg.Data)
	case models.MessageTypeTyping:
		c.handleTyping(msg.Data)
	default:
		c.SendError("unknown message type")
	}
}

func (c *Client) handleListSubscribe(data json.RawMessage) {
	var payload struct {
		UUIDs []string `json:"uuids"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		c.SendError("invalid list_subscribe payload")
		return
	}
	if len(payload.UUIDs) > maxListSubUUIDs {
		payload.UUIDs = payload.UUIDs[:maxListSubUUIDs]
	}
	authorized, err := c.Hub.conversationStore.FilterAuthorizedListUUIDs(c.ID, payload.UUIDs)
	if err != nil {
		return
	}
	c.Hub.SubscribeListReplace(c, authorized)
}

// handleConversationSubscribe registers the open-conversation sub; authz is enforced because content (not just typing) flows through it.
func (c *Client) handleConversationSubscribe(data json.RawMessage) {
	var subscribeMsg models.ConversationSubscribe
	if err := json.Unmarshal(data, &subscribeMsg); err != nil {
		c.SendError("invalid subscription format")
		return
	}

	if subscribeMsg.ConversationUUID == "" {
		c.SendError("conversation_uuid is required")
		return
	}

	// Authz: silently reject if the agent can't read this conversation.
	authorized, err := c.Hub.conversationStore.FilterAuthorizedListUUIDs(c.ID, []string{subscribeMsg.ConversationUUID})
	if err != nil || len(authorized) == 0 {
		return
	}

	c.Hub.SubscribeOpenConv(c, subscribeMsg.ConversationUUID)
}

// handleTyping handles typing indicator messages.
//
// Same trust assumption as handleConversationSubscribe: the sender is an
// authenticated agent. A hostile agent could broadcast fake typing to any
// conversation UUID (including widget clients), but typing is ephemeral and
// cosmetic; adding per-frame authz isn't worth the DB cost today.
func (c *Client) handleTyping(data json.RawMessage) {
	var typingMsg models.TypingMessage
	if err := json.Unmarshal(data, &typingMsg); err != nil {
		c.SendError("invalid typing format")
		return
	}

	if typingMsg.ConversationUUID == "" {
		c.SendError("conversation_uuid is required for typing")
		return
	}

	c.Hub.BroadcastTypingToConversation(typingMsg.ConversationUUID, typingMsg)
}

// close closes the client connection.
func (c *Client) close() {
	c.Closed.Set(true)
	close(c.Send)
}

// SendError sends an error message to client.
func (c *Client) SendError(msg string) {
	out := models.Message{
		Type: models.MessageTypeError,
		Data: msg,
	}
	b, _ := json.Marshal(out)

	select {
	case c.Send <- models.WSMessage{Data: b, MessageType: websocket.TextMessage}:
	default:
		c.Hub.lo.Warn("client send channel full, could not send error message", "client_id", c.ID)
		c.Hub.RemoveClient(c)
		c.close()
	}
}

// SendMessage sends a message to client.
func (c *Client) SendMessage(b []byte, typ byte) {
	if c.Closed.Get() {
		c.Hub.lo.Warn("attempted to send message to closed client", "client_id", c.ID)
		return
	}
	select {
	case c.Send <- models.WSMessage{Data: b, MessageType: websocket.TextMessage}:
	default:
	}
}
