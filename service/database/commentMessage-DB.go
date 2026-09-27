package database

import (
	"database/sql"
	"errors"
)

// ReactionCreatedResult matches the ReactionCreated schema in the OpenAPI spec.
type ReactionCreatedResult struct {
	Username  string `json:"username"`
	Emoticon  string `json:"emoticon"`
	MessageID int64  `json:"messageId"`
}

// MessageExists checks whether a message exists and belongs to the given conversation.
func (db *appdbimpl) MessageExists(messageID int64, conversationID int64) (bool, error) {
	var found int64
	err := db.c.QueryRow(`
		SELECT id FROM Message
		WHERE id = ? AND chat = ?
	`, messageID, conversationID).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// UpsertReaction adds a reaction to a message, or updates it if the user
// has already reacted. Uses INSERT OR REPLACE which in SQLite deletes the
// existing row with the same PRIMARY KEY and inserts a new one.
func (db *appdbimpl) UpsertReaction(messageID int64, username string, emoticon string) (ReactionCreatedResult, error) {
	var result ReactionCreatedResult

	_, err := db.c.Exec(`
		INSERT OR REPLACE INTO reaction (message, userOfReaction, emoticon)
		VALUES (?, ?, ?)
	`, messageID, username, emoticon)
	if err != nil {
		return result, err
	}

	result.Username = username
	result.Emoticon = emoticon
	result.MessageID = messageID

	return result, nil
}
