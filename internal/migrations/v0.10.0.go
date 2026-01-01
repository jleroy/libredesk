package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V0_10_0 updates the database schema to v0.10.0.
func V0_10_0(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	_, err := db.Exec(`
		ALTER TABLE conversations
		ADD COLUMN IF NOT EXISTS last_interaction TEXT NULL,
		ADD COLUMN IF NOT EXISTS last_interaction_sender message_sender_type NULL,
		ADD COLUMN IF NOT EXISTS last_interaction_at TIMESTAMPTZ NULL;

		CREATE INDEX IF NOT EXISTS index_conversations_on_last_interaction_at
		ON conversations(last_interaction_at);
	`)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS conversation_mentions (
			id BIGSERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			conversation_id BIGINT REFERENCES conversations(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
			message_id BIGINT REFERENCES conversation_messages(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
			mentioned_user_id BIGINT REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE,
			mentioned_team_id INT REFERENCES teams(id) ON DELETE CASCADE ON UPDATE CASCADE,
			mentioned_by_user_id BIGINT REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
			CONSTRAINT constraint_mention_target CHECK (
				(mentioned_user_id IS NOT NULL AND mentioned_team_id IS NULL) OR
				(mentioned_user_id IS NULL AND mentioned_team_id IS NOT NULL)
			)
		);

		CREATE INDEX IF NOT EXISTS index_conversation_mentions_on_mentioned_user_id ON conversation_mentions(mentioned_user_id);
		CREATE INDEX IF NOT EXISTS index_conversation_mentions_on_mentioned_team_id ON conversation_mentions(mentioned_team_id);
		CREATE INDEX IF NOT EXISTS index_conversation_mentions_on_conversation_id ON conversation_mentions(conversation_id);
	`)
	if err != nil {
		return err
	}

	// Add email notification template for mentions
	_, err = db.Exec(`
		DO $$
		BEGIN
			IF NOT EXISTS (SELECT 1 FROM templates WHERE "name" = 'Mentioned in conversation') THEN
				INSERT INTO templates
					("type", body, is_default, "name", subject, is_builtin)
					VALUES (
					'email_notification'::template_type,
'<p>{{ .Author.FullName }} mentioned you in a private note on conversation #{{ .Conversation.ReferenceNumber }}.</p>

<p>
<a href="{{ RootURL }}/inboxes/mentioned/conversation/{{ .Conversation.UUID }}?scrollTo={{ .Message.UUID }}">View Conversation</a>
</p>

<p>
Best regards,<br>
Libredesk
</p>',
					false,
					'Mentioned in conversation',
					'{{ .Author.FullName }} mentioned you in conversation #{{ .Conversation.ReferenceNumber }}',
					true
				);
			END IF;
		END$$;
	`)
	if err != nil {
		return err
	}

	return nil
}
