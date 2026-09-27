package database

import (
	"database/sql"
	"errors"
)

// IsMessageSender checks whether the given user is the sender of the message.
func (db *appdbimpl) IsMessageSender(messageID int64, username string) (bool, error) {
	var sender string
	err := db.c.QueryRow(`
		SELECT sender FROM Message WHERE id = ?
	`, messageID).Scan(&sender)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return sender == username, nil
}

// DeleteMessage removes the message from the database.
// Thanks to the ON DELETE CASCADE foreign keys defined in the schema,
// deleting a message automatically removes its related rows
// in messageStatus and reaction.
func (db *appdbimpl) DeleteMessage(messageID int64) error {
	res, err := db.c.Exec(`
		DELETE FROM Message WHERE id = ?
	`, messageID)
	if err != nil {
		return err
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("message not found")
	}

	return nil
}
