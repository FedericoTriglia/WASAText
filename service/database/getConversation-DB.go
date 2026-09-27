package database

import (
	"database/sql"
	"errors"
)

// The structs below mirror the "Conversation" schema defined in the OpenAPI spec.

type Conversation struct {
	ID           int64     `json:"id"`
	Type         string    `json:"type"` // "direct" or "group"
	Name         string    `json:"name"`
	PhotoURL     string    `json:"photoUrl,omitempty"`
	Participants []string  `json:"participants"`
	Messages     []Message `json:"messages"`
}

type Message struct {
	ID             int64      `json:"id"`
	SenderUsername string     `json:"senderUsername"`
	Content        string     `json:"content,omitempty"`
	PhotoURL       string     `json:"photoUrl,omitempty"`
	Timestamp      string     `json:"timestamp"`
	Status         string     `json:"status"`
	ReplyTo        *ReplyTo   `json:"replyTo,omitempty"`
	Reactions      []Reaction `json:"reactions"`
	IsForwarded    bool       `json:"isForwarded"`
}

type ReplyTo struct {
	MessageID      int64  `json:"messageId"`
	Content        string `json:"content,omitempty"`
	SenderUsername string `json:"senderUsername"`
}

type Reaction struct {
	Username  string `json:"username"`
	Emoticon  string `json:"emoticon"`
	Timestamp string `json:"timestamp"`
}

// ConversationExists checks whether a conversation exists in the database.
func (db *appdbimpl) ConversationExists(conversationID int64) (bool, error) {
	var id int64
	err := db.c.QueryRow(`
		SELECT id FROM Chat WHERE id = ?
	`, conversationID).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// IsParticipant checks whether a user is a participant in a given conversation.
func (db *appdbimpl) IsParticipant(conversationID int64, username string) (bool, error) {
	var found string
	err := db.c.QueryRow(`
		SELECT userOfChat FROM usChat
		WHERE chat = ? AND userOfChat = ?
	`, conversationID, username).Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// GetConversation returns the full conversation including all messages.
func (db *appdbimpl) GetConversation(conversationID int64, username string) (Conversation, error) {
	conv := Conversation{ID: conversationID}

	// 1. Determine the type (direct/group), name, and photo of the conversation
	var chatName string
	var photoURL sql.NullString
	err := db.c.QueryRow(`
		SELECT chatName, photo_url FROM GroupChat WHERE chat = ?
	`, conversationID).Scan(&chatName, &photoURL)

	if errors.Is(err, sql.ErrNoRows) {
		// Direct chat — look up the other participant's username and photo
		conv.Type = "direct"
		var otherUsername string
		var otherPhoto sql.NullString
		err2 := db.c.QueryRow(`
			SELECT u.username, u.photo_url
			FROM User u, usChat uc
			WHERE uc.chat = ?
			  AND uc.userOfChat = u.username
			  AND uc.userOfChat != ?
		`, conversationID, username).Scan(&otherUsername, &otherPhoto)
		if err2 != nil {
			return conv, err2
		}
		conv.Name = otherUsername
		if otherPhoto.Valid {
			conv.PhotoURL = otherPhoto.String
		}
	} else if err != nil {
		return conv, err
	} else {
		// Group chat
		conv.Type = "group"
		conv.Name = chatName
		if photoURL.Valid {
			conv.PhotoURL = photoURL.String
		}
	}

	// 2. Retrieve all participants
	participants, err := db.getParticipants(conversationID)
	if err != nil {
		return conv, err
	}
	conv.Participants = participants

	// 3. Retrieve messages in reverse chronological order
	messages, err := db.getMessages(conversationID, username)
	if err != nil {
		return conv, err
	}
	conv.Messages = messages

	return conv, nil
}

// getParticipants returns the list of usernames of all conversation participants.
func (db *appdbimpl) getParticipants(conversationID int64) ([]string, error) {
	rows, err := db.c.Query(`
		SELECT userOfChat FROM usChat WHERE chat = ?
	`, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var participants []string
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, err
		}
		participants = append(participants, p)
	}
	return participants, rows.Err()
}

// getMessages retrieves all messages for a conversation in reverse chronological order
// (most recent first), including status, replyTo, and reactions for each message.
func (db *appdbimpl) getMessages(conversationID int64, username string) ([]Message, error) {
	rows, err := db.c.Query(`
		SELECT id, sender, format, text, image_url, instant, answers, forwards
		FROM Message
		WHERE chat = ?
		ORDER BY instant DESC
	`, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var messages []Message
	for rows.Next() {
		var msg Message
		var text sql.NullString
		var imageURL sql.NullString
		var format string
		var answersID sql.NullInt64  // ID of the message this one replies to (nullable)
		var forwardsID sql.NullInt64 // ID of the original message if this one was forwarded

		if err := rows.Scan(
			&msg.ID, &msg.SenderUsername, &format,
			&text, &imageURL, &msg.Timestamp, &answersID, &forwardsID,
		); err != nil {
			return nil, err
		}

		msg.IsForwarded = forwardsID.Valid

		if format == "IMAGE" {
			msg.PhotoURL = imageURL.String
		} else {
			msg.Content = text.String
		}

		// Status: read from messageStatus for the current user.
		// If the message was sent by the current user, return the "worst" status
		// across all recipients (sent < received < read).
		status, err := db.getMessageStatus(msg.ID, msg.SenderUsername, username)
		if err != nil {
			return nil, err
		}
		msg.Status = status

		// ReplyTo: if this message is a reply, fetch the original message data
		if answersID.Valid {
			replyTo, err := db.getReplyTo(answersID.Int64)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return nil, err
			}
			msg.ReplyTo = replyTo
		}

		// Reactions: fetch all reactions for this message
		reactions, err := db.getReactions(msg.ID)
		if err != nil {
			return nil, err
		}
		msg.Reactions = reactions

		messages = append(messages, msg)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Ensure the messages field is always a JSON array (never null)
	if messages == nil {
		messages = []Message{}
	}

	return messages, nil
}

// getMessageStatus returns the message status from the current user's perspective:
//   - if the current user is the sender: returns the "worst" status across all recipients
//     (sent if at least one hasn't received it, received if all received but not all read,
//     read only if all recipients have read it)
//   - if the current user is a recipient: returns their personal status
func (db *appdbimpl) getMessageStatus(messageID int64, senderUsername string, currentUsername string) (string, error) {
	if senderUsername == currentUsername {
		// Sender view: compute the minimum status across all recipients
		var status string
		err := db.c.QueryRow(`
			SELECT CASE
				WHEN EXISTS (
					SELECT 1 FROM messageStatus
					WHERE message = ? AND status = 'sent'
				) THEN 'sent'
				WHEN EXISTS (
					SELECT 1 FROM messageStatus
					WHERE message = ? AND status = 'received'
				) THEN 'received'
				ELSE 'read'
			END
		`, messageID, messageID).Scan(&status)
		return status, err
	}

	// Recipient view: return this user's personal status
	var status string
	err := db.c.QueryRow(`
		SELECT status FROM messageStatus
		WHERE message = ? AND userOfMessageStatus = ?
	`, messageID, currentUsername).Scan(&status)
	if errors.Is(err, sql.ErrNoRows) {
		return "sent", nil // default if no status row exists yet
	}
	return status, err
}

// getReplyTo retrieves the original message that this one is replying to.
func (db *appdbimpl) getReplyTo(messageID int64) (*ReplyTo, error) {
	var replyTo ReplyTo
	var text sql.NullString
	var imageURL sql.NullString
	var format string

	err := db.c.QueryRow(`
		SELECT id, sender, format, text, image_url
		FROM Message
		WHERE id = ?
	`, messageID).Scan(&replyTo.MessageID, &replyTo.SenderUsername, &format, &text, &imageURL)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, err
	}

	if format == "IMAGE" {
		replyTo.Content = imageURL.String
	} else {
		replyTo.Content = text.String
	}

	return &replyTo, nil
}

// getReactions retrieves all reactions for a given message.
func (db *appdbimpl) getReactions(messageID int64) ([]Reaction, error) {
	rows, err := db.c.Query(`
		SELECT userOfReaction, emoticon
		FROM reaction
		WHERE message = ?
	`, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var reactions []Reaction
	for rows.Next() {
		var react Reaction
		if err := rows.Scan(&react.Username, &react.Emoticon); err != nil {
			return nil, err
		}
		// The reaction table has no timestamp column in the database schema,
		// so the field is left empty
		reactions = append(reactions, react)
	}

	if reactions == nil {
		reactions = []Reaction{}
	}

	return reactions, rows.Err()
}
