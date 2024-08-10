package models

import "time"

type Team struct {
	ID                      int       `db:"id" json:"id"`
	CreatedAt               time.Time `db:"created_at" json:"created_at"`
	UpdatedAt               time.Time `db:"updated_at" json:"updated_at"`
	Name                    string    `db:"name" json:"name"`
	AutoAssignConversations bool      `db:"auto_assign_conversations" json:"auto_assign_conversations"`
}
