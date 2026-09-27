package database

// MarkMessagesRead updates to 'read' all messages in the conversation
// that are still in 'sent' or 'received' state for the current user.
// Messages sent by the user themselves are excluded (the sender has
// no messageStatus row for their own messages).
func (db *appdbimpl) MarkMessagesRead(conversationID int64, username string) error {
	_, err := db.c.Exec(`
		UPDATE messageStatus
		SET status = 'read'
		WHERE status != 'read'
		  AND userOfMessageStatus = ?
		  AND message IN (
		  	SELECT id FROM Message WHERE chat = ?
		  )
	`, username, conversationID)
	return err
}
