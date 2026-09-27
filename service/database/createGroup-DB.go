package database

import (
	"database/sql"
	"errors"
)

// CreateGroupResult matches the 201 response schema for createGroup in the OpenAPI spec.
type CreateGroupResult struct {
	GroupID   int64    `json:"groupId"`
	Name      string   `json:"name"`
	Members   []string `json:"members"`
	CreatedAt string   `json:"createdAt"`
}

// UserExists checks whether a user exists in the database.
func (db *appdbimpl) UserExists(username string) (bool, error) {
	var found string
	err := db.c.QueryRow(`
		SELECT username FROM User WHERE username = ?
	`, username).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// CreateGroup creates a new group with the given members and returns the group data.
// The entire operation runs inside a transaction: if any step fails,
// everything is rolled back and the database remains unchanged.
func (db *appdbimpl) CreateGroup(name string, members []string) (CreateGroupResult, error) {
	var result CreateGroupResult

	// Open a transaction: Chat, GroupChat, and usChat must all be inserted
	// together — if any insert fails the whole group creation is rolled back
	tx, err := db.c.Begin()
	if err != nil {
		return result, err
	}
	// If anything goes wrong, roll back all changes
	defer func() { _ = tx.Rollback() }()

	// 1. Insert a new row into Chat and retrieve the generated ID
	res, err := tx.Exec(`INSERT INTO Chat DEFAULT VALUES`)
	if err != nil {
		return result, err
	}
	chatID, err := res.LastInsertId()
	if err != nil {
		return result, err
	}

	// 2. Insert the group into GroupChat using the new chat ID
	_, err = tx.Exec(`
		INSERT INTO GroupChat (chat, chatName, photo_url)
		VALUES (?, ?, NULL)
	`, chatID, name)
	if err != nil {
		return result, err
	}

	// 3. Insert every member into usChat
	for _, member := range members {
		_, err = tx.Exec(`
			INSERT INTO usChat (chat, userOfChat) VALUES (?, ?)
		`, chatID, member)
		if err != nil {
			return result, err
		}
	}

	// 4. Retrieve the creation timestamp
	//    (Chat has no instant column, so SQLite's datetime('now') is used instead)
	var createdAt string
	err = tx.QueryRow(`SELECT datetime('now')`).Scan(&createdAt)
	if err != nil {
		return result, err
	}

	// 5. All steps succeeded — commit the transaction
	if err := tx.Commit(); err != nil {
		return result, err
	}

	result.GroupID = chatID
	result.Name = name
	result.Members = members
	result.CreatedAt = createdAt

	return result, nil
}
