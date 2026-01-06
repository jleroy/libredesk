package conversation

import (
	"encoding/json"
	"time"

	cmodels "github.com/abhinavxd/libredesk/internal/conversation/models"
	wsmodels "github.com/abhinavxd/libredesk/internal/ws/models"
)

// BroadcastNewMessage broadcasts a new message to all users.
func (m *Manager) BroadcastNewMessage(message *cmodels.Message) {
	m.broadcastToUsers([]int{}, wsmodels.Message{
		Type: wsmodels.MessageTypeNewMessage,
		Data: map[string]interface{}{
			"conversation_uuid": message.ConversationUUID,
			"content":           message.TextContent,
			"created_at":        message.CreatedAt.Format(time.RFC3339),
			"uuid":              message.UUID,
			"private":           message.Private,
			"type":              message.Type,
			"sender_type":       message.SenderType,
		},
	})
}

// BroadcastMessageUpdate broadcasts a message update to all users.
func (m *Manager) BroadcastMessageUpdate(conversationUUID, messageUUID, prop string, value any) {
	message := wsmodels.Message{
		Type: wsmodels.MessageTypeMessagePropUpdate,
		Data: map[string]interface{}{
			"conversation_uuid": conversationUUID,
			"uuid":              messageUUID,
			"prop":              prop,
			"value":             value,
		},
	}
	m.broadcastToUsers([]int{}, message)
}

// BroadcastConversationUpdate broadcasts a conversation update to all users.
func (m *Manager) BroadcastConversationUpdate(conversationUUID, prop string, value any) {
	message := wsmodels.Message{
		Type: wsmodels.MessageTypeConversationPropertyUpdate,
		Data: map[string]interface{}{
			"uuid":  conversationUUID,
			"prop":  prop,
			"value": value,
		},
	}
	m.broadcastToUsers([]int{}, message)
}

// broadcastToUsers broadcasts a message to a list of users, if the list is empty it broadcasts to all users.
func (m *Manager) broadcastToUsers(userIDs []int, message wsmodels.Message) {
	messageBytes, err := json.Marshal(message)
	if err != nil {
		m.lo.Error("error marshalling WS message", "error", err)
		return
	}
	m.wsHub.BroadcastMessage(wsmodels.BroadcastMessage{
		Data:  messageBytes,
		Users: userIDs,
	})
}
