package models

// Action constants for WebSocket messages.
const (
	MessageTypeMessagePropUpdate          = "message_prop_update"
	MessageTypeConversationPropertyUpdate = "conversation_prop_update"
	MessageTypeNewMessage                 = "new_message"
	MessageTypeNewConversation            = "new_conversation"
	MessageTypeNewNotification            = "new_notification"
	MessageTypeError                      = "error"
)

// WSMessage represents a WS message.
type WSMessage struct {
	MessageType int
	Data        []byte
}

// Message represents a WebSocket message to be sent.
type Message struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// BroadcastMessage represents a message to be pushed to users.
type BroadcastMessage struct {
	Data  []byte `json:"data"`
	Users []int  `json:"users"`
}
