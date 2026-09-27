package database

// LeaveGroup removes the user from the group.
// If the user was the last member, the entire chat is deleted from the Chat table;
// thanks to the ON DELETE CASCADE foreign keys, this automatically removes
// GroupChat, usChat, all Messages, and their related rows in messageStatus and reaction.
func (db *appdbimpl) LeaveGroup(groupID int64, username string) error {
	tx, err := db.c.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// 1. Remove the user from the usChat table
	_, err = tx.Exec(`
		DELETE FROM usChat
		WHERE chat = ? AND userOfChat = ?
	`, groupID, username)
	if err != nil {
		return err
	}

	// 2. Count the remaining members
	var remaining int
	err = tx.QueryRow(`
		SELECT COUNT(*) FROM usChat WHERE chat = ?
	`, groupID).Scan(&remaining)
	if err != nil {
		return err
	}

	// 3. If no members remain, delete the entire chat (cascade removes all related data)
	if remaining == 0 {
		_, err = tx.Exec(`
			DELETE FROM Chat WHERE id = ?
		`, groupID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}
