package notifier

import (
	"encoding/json"

	"github.com/abhinavxd/libredesk/internal/notification/models"
	wsmodels "github.com/abhinavxd/libredesk/internal/ws/models"
	"github.com/volatiletech/null/v9"
	"github.com/zerodha/logf"
)

// WSHub defines the interface for the WebSocket hub.
type WSHub interface {
	BroadcastMessage(msg wsmodels.BroadcastMessage)
}

// Notification represents a notification to be sent through all channels.
type Notification struct {
	// Core notification fields
	Type           models.NotificationType
	RecipientIDs   []int
	Title          string
	Body           null.String
	ConversationID null.Int
	MessageID      null.Int
	ActorID        null.Int
	Meta           json.RawMessage

	// For WebSocket broadcast
	ConversationUUID string
	ActorFirstName   string
	ActorLastName    string

	// Email fields (optional - if empty, no email sent)
	Email *EmailNotification
}

// EmailNotification holds email channel notification details.
type EmailNotification struct {
	Recipients []string
	Subject    string
	Content    string
}

// Dispatcher coordinates sending notifications through multiple channels: WS, DB, email.
type Dispatcher struct {
	inApp    *UserNotificationManager
	outbound *Service
	wsHub    WSHub
	lo       *logf.Logger
}

// DispatcherOpts contains options for creating a new Dispatcher.
type DispatcherOpts struct {
	InApp    *UserNotificationManager
	Outbound *Service
	WSHub    WSHub
	Lo       *logf.Logger
}

// NewDispatcher creates a new notification Dispatcher.
func NewDispatcher(opts DispatcherOpts) *Dispatcher {
	return &Dispatcher{
		inApp:    opts.InApp,
		outbound: opts.Outbound,
		wsHub:    opts.WSHub,
		lo:       opts.Lo,
	}
}

// broadcastNotification broadcasts a notification via WebSocket to specified users.
func (d *Dispatcher) broadcastNotification(userIDs []int, notification any) {
	if d.wsHub == nil {
		return
	}
	message := wsmodels.Message{
		Type: wsmodels.MessageTypeNewNotification,
		Data: notification,
	}
	msgB, err := json.Marshal(message)
	if err != nil {
		d.lo.Error("error marshalling notification for WebSocket", "error", err)
		return
	}
	d.wsHub.BroadcastMessage(wsmodels.BroadcastMessage{
		Data:  msgB,
		Users: userIDs,
	})
}

// Send sends a notification through all configured channels.
// For each recipient:
// 1. Creates in-app notification (DB)
// 2. Broadcasts via WebSocket
// 3. Sends email if Email field is provided and recipient has email
func (d *Dispatcher) Send(n Notification) {
	for i, recipientID := range n.RecipientIDs {
		// 1. Create in-app notification
		notification, err := d.inApp.Create(
			recipientID,
			n.Type,
			n.Title,
			n.Body,
			n.ConversationID,
			n.MessageID,
			n.ActorID,
			n.Meta,
		)
		if err != nil {
			d.lo.Error("error creating in-app notification",
				"recipient_id", recipientID,
				"type", n.Type,
				"error", err)
		} else {
			// 2. Broadcast via WebSocket
			notification.ConversationUUID = null.StringFrom(n.ConversationUUID)
			notification.ActorFirstName = null.StringFrom(n.ActorFirstName)
			notification.ActorLastName = null.StringFrom(n.ActorLastName)
			d.broadcastNotification([]int{recipientID}, notification)
		}

		// 3. Send email
		if d.outbound != nil && n.Email != nil {
			// Get recipient email - either from Email.Recipients array or skip
			var recipientEmail string
			if len(n.Email.Recipients) > i {
				recipientEmail = n.Email.Recipients[i]
			} else if len(n.Email.Recipients) == 1 {
				// Single email for all recipients (broadcast case)
				recipientEmail = n.Email.Recipients[0]
			}

			if recipientEmail != "" {
				if err := d.outbound.Send(Message{
					RecipientEmails: []string{recipientEmail},
					Subject:         n.Email.Subject,
					Content:         n.Email.Content,
					Provider:        ProviderEmail,
				}); err != nil {
					d.lo.Error("error sending email notification",
						"recipient_id", recipientID,
						"email", recipientEmail,
						"type", n.Type,
						"error", err)
				}
			}
		}
	}
}

// SendWithEmails sends notifications where each recipient has their own email content.
// This is useful when email content is personalized per recipient.
func (d *Dispatcher) SendWithEmails(n Notification, emails []EmailNotification) {
	for i, recipientID := range n.RecipientIDs {
		// 1. Create in-app notification
		notification, err := d.inApp.Create(
			recipientID,
			n.Type,
			n.Title,
			n.Body,
			n.ConversationID,
			n.MessageID,
			n.ActorID,
			n.Meta,
		)
		if err != nil {
			d.lo.Error("error creating in-app notification",
				"recipient_id", recipientID,
				"type", n.Type,
				"error", err)
			// Continue to try email even if DB insert failed
		} else {
			// 2. Broadcast via WebSocket (only if DB succeeded - need notification object)
			notification.ConversationUUID = null.StringFrom(n.ConversationUUID)
			notification.ActorFirstName = null.StringFrom(n.ActorFirstName)
			notification.ActorLastName = null.StringFrom(n.ActorLastName)
			d.broadcastNotification([]int{recipientID}, notification)
		}

		// 3. Send email
		if d.outbound != nil && i < len(emails) && len(emails[i].Recipients) > 0 {
			email := emails[i]
			if err := d.outbound.Send(Message{
				RecipientEmails: email.Recipients,
				Subject:         email.Subject,
				Content:         email.Content,
				Provider:        ProviderEmail,
			}); err != nil {
				d.lo.Error("error sending email notification",
					"recipient_id", recipientID,
					"type", n.Type,
					"error", err)
			}
		}
	}
}
