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
	return err
}
