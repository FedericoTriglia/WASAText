package database

import "errors"

// DeleteReaction removes the user's reaction from the specified message.
// Returns "reaction not found" if the user had no reaction on that message.
func (db *appdbimpl) DeleteReaction(messageID int64, username string) error {
	res, err := db.c.Exec(`
		DELETE FROM reaction
		WHERE message = ? AND userOfReaction = ?
	`, messageID, username)
	if err != nil {
		return err
	}

	// RowsAffected() tells us how many rows were deleted.
	// If 0 rows were affected, the reaction did not exist.
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("reaction not found")
	}

	return nil
}
