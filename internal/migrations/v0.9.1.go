package migrations

import (
	"github.com/jmoiron/sqlx"
	"github.com/knadh/koanf/v2"
	"github.com/knadh/stuffbin"
)

// V0_9_1 updates the database schema to v0.9.1.
func V0_9_1(db *sqlx.DB, fs stuffbin.FileSystem, ko *koanf.Koanf) error {
	// Create conversation_drafts table and index if they don't exist
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS conversation_drafts (
			id BIGSERIAL PRIMARY KEY,
			created_at TIMESTAMPTZ DEFAULT NOW(),
			updated_at TIMESTAMPTZ DEFAULT NOW(),
			conversation_id BIGINT REFERENCES conversations(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
			user_id BIGINT REFERENCES users(id) ON DELETE CASCADE ON UPDATE CASCADE NOT NULL,
			content TEXT NOT NULL,
			meta JSONB DEFAULT '{}'::jsonb NOT NULL
		);

		CREATE UNIQUE INDEX IF NOT EXISTS index_uniq_conversation_drafts_on_conversation_id_and_user_id 
		ON conversation_drafts (conversation_id, user_id);
	`)
	if err != nil {
		return err
	}

	return nil
}
