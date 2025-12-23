package conversation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/abhinavxd/libredesk/internal/conversation/models"
	"github.com/abhinavxd/libredesk/internal/envelope"
)

// UpsertConversationDraft saves or updates a draft for a conversation.
func (m *Manager) UpsertConversationDraft(conversationID, userID int, content string, meta json.RawMessage) (models.ConversationDraft, error) {
	var draft models.ConversationDraft

	if err := m.q.UpsertConversationDraft.Get(&draft, conversationID, userID, content, meta); err != nil {
		m.lo.Error("error upserting conversation draft", "conversation_id", conversationID, "user_id", userID, "error", err)
		return draft, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorUpdating", "name", "draft"), nil)
	}

	return draft, nil
}

// GetConversationDraft retrieves a draft for a conversation by ID or UUID.
func (m *Manager) GetConversationDraft(conversationID int, uuid string, userID int) (models.ConversationDraft, error) {
	var draft models.ConversationDraft
	var uuidParam any
	if uuid != "" {
		uuidParam = uuid
	}

	if err := m.q.GetConversationDraft.Get(&draft, conversationID, uuidParam, userID); err != nil {
		if err == sql.ErrNoRows {
			return draft, nil
		}
		m.lo.Error("error fetching conversation draft", "conversation_id", conversationID, "uuid", uuid, "user_id", userID, "error", err)
		return draft, envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorFetching", "name", "draft"), nil)
	}

	return draft, nil
}

// DeleteConversationDraft deletes a draft for a conversation by ID or UUID.
func (m *Manager) DeleteConversationDraft(conversationID int, uuid string, userID int) error {
	var uuidParam any
	if uuid != "" {
		uuidParam = uuid
	}

	if _, err := m.q.DeleteConversationDraft.Exec(conversationID, uuidParam, userID); err != nil {
		m.lo.Error("error deleting conversation draft", "conversation_id", conversationID, "uuid", uuid, "user_id", userID, "error", err)
		return envelope.NewError(envelope.GeneralError, m.i18n.Ts("globals.messages.errorDeleting", "name", "draft"), nil)
	}

	return nil
}

// DeleteStaleDrafts deletes drafts older than the specified retention period.
func (m *Manager) DeleteStaleDrafts(ctx context.Context, retentionPeriod time.Duration) error {
	// Format duration as PostgreSQL interval string
	intervalStr := fmt.Sprintf("%d seconds", int(retentionPeriod.Seconds()))

	res, err := m.q.DeleteStaleDrafts.ExecContext(ctx, intervalStr)
	if err != nil {
		m.lo.Error("error deleting stale drafts", "error", err)
		return err
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected > 0 {
		m.lo.Info("deleted stale drafts", "count", rowsAffected)
	}

	return nil
}
