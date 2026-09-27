package database

import (
	"database/sql"
	"errors"
)

// UsernameAddedToGroupResult matches the UsernameAddedToGroup schema in the OpenAPI spec.
type UsernameAddedToGroupResult struct {
	GroupID  int64  `json:"groupId"`
	Username string `json:"username"`
	AddedAt  string `json:"addedAt"`
}

// GroupExists checks whether a group exists by looking it up in the GroupChat table.
func (db *appdbimpl) GroupExists(groupID int64) (bool, error) {
	var found int64
	err := db.c.QueryRow(`
		SELECT chat FROM GroupChat WHERE chat = ?
	`, groupID).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// AddUserToGroup inserts a new member into the usChat table
// and returns the result of the operation.
func (db *appdbimpl) AddUserToGroup(groupID int64, username string) (UsernameAddedToGroupResult, error) {
	var result UsernameAddedToGroupResult

	_, err := db.c.Exec(`
		INSERT INTO usChat (chat, userOfChat) VALUES (?, ?)
	`, groupID, username)
	if err != nil {
		return result, err
	}

	// Retrieve the current timestamp to use as addedAt
	var addedAt string
	err = db.c.QueryRow(`SELECT datetime('now')`).Scan(&addedAt)
	if err != nil {
		return result, err
	}

	result.GroupID = groupID
	result.Username = username
	result.AddedAt = addedAt

	return result, nil
}
