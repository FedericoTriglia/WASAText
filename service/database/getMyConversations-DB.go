package database

import (
	"database/sql"
	"errors"
)

// ConversationPreview matches a single element of the Conversations array
// defined in the OpenAPI spec.
type ConversationPreview struct {
	ID          int64               `json:"id"`
	Type        string              `json:"type"` // "direct" or "group"
	Name        string              `json:"name"`
	PhotoURL    string              `json:"photoUrl"`
	LastMessage *LastMessagePreview `json:"lastMessage,omitempty"`
	UnreadCount int                 `json:"unreadCount"`
}

// LastMessagePreview matches the lastMessage sub-object in the OpenAPI spec.
type LastMessagePreview struct {
	Preview   string `json:"preview"`
	Timestamp string `json:"timestamp"`
	IsPhoto   bool   `json:"isPhoto"`
}

// GetMyConversations returns all conversations for the user,
// ordered by the timestamp of the last message (most recent first).
func (db *appdbimpl) GetMyConversations(username string) ([]ConversationPreview, error) {

	// Retrieve all chat IDs the user participates in,
	// ordered by the timestamp of the last message in each chat.
	// The subquery (SELECT MAX(instant) ...) finds the latest message timestamp
	// per chat, which ORDER BY uses for sorting.
	// Chats with no messages appear last (NULL sorts last with DESC).
	rows, err := db.c.Query(`
		SELECT chat
		FROM usChat
		WHERE userOfChat = ?
		ORDER BY (
			SELECT MAX(instant)
			FROM Message
			WHERE Message.chat = usChat.chat
		) DESC
	`, username)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	// Mark as 'received' all messages that are still 'sent' for this user —
	// loading the conversation list means the user has seen the notification
	if err := db.markMessagesReceived(username); err != nil {
		return nil, err
	}

	var conversations []ConversationPreview

	for rows.Next() {
		var chatID int64
		if err := rows.Scan(&chatID); err != nil {
			return nil, err
		}

		conv, err := db.buildConversationPreview(chatID, username)
		if err != nil {
			return nil, err
		}

		conversations = append(conversations, conv)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return conversations, nil
}

// getLastMessage retrieves a preview of the last message in a chat.
// Returns nil, sql.ErrNoRows if the chat has no messages.
func (db *appdbimpl) getLastMessage(chatID int64) (*LastMessagePreview, error) {
	var preview LastMessagePreview
	var text sql.NullString
	var imageURL sql.NullString
	var format string

	err := db.c.QueryRow(`
		SELECT format, text, image_url, instant
		FROM Message
		WHERE chat = ?
		ORDER BY instant DESC
		LIMIT 1
	`, chatID).Scan(&format, &text, &imageURL, &preview.Timestamp)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, err
	}

	// For image messages the preview is the image URL and isPhoto is true;
	// for text messages the preview is the text truncated to 100 characters.
	if format == "IMAGE" {
		preview.IsPhoto = true
		preview.Preview = imageURL.String
	} else {
		preview.IsPhoto = false
		content := text.String
		if len(content) > 100 {
			content = content[:100]
		}
		preview.Preview = content
	}

	return &preview, nil
}

// getUnreadCount counts the messages that the user has not yet read in a conversation.
// A message is considered unread if its status in messageStatus is 'sent' or 'received'
// (i.e. NOT 'read') for that user.
func (db *appdbimpl) getUnreadCount(chatID int64, username string) (int, error) {
	var count int
	err := db.c.QueryRow(`
		SELECT COUNT(*)
		FROM Message m
		JOIN messageStatus ms ON ms.message = m.id
		WHERE m.chat = ?
		  AND ms.userOfMessageStatus = ?
		  AND ms.status != 'read'
		  AND m.sender != ?
	`, chatID, username, username).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// buildConversationPreview builds the ConversationPreview object for a single chat.
func (db *appdbimpl) buildConversationPreview(chatID int64, username string) (ConversationPreview, error) {
	conv := ConversationPreview{ID: chatID}

	// Determine whether the chat is a group by looking it up in GroupChat
	var chatName string
	var photoURL sql.NullString
	err := db.c.QueryRow(`
		SELECT chatName, photo_url
		FROM GroupChat
		WHERE chat = ?
	`, chatID).Scan(&chatName, &photoURL)

	if errors.Is(err, sql.ErrNoRows) {
		// Not found in GroupChat — this is a direct (private) chat
		conv.Type = "direct"

		// Retrieve the other participant's username and photo
		var otherUsername string
		var otherPhoto sql.NullString
		err2 := db.c.QueryRow(`
			SELECT u.username, u.photo_url
			FROM User u, usChat uc
			WHERE uc.chat = ?
			  AND uc.userOfChat = u.username
			  AND uc.userOfChat != ?
		`, chatID, username).Scan(&otherUsername, &otherPhoto)
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
		// Found in GroupChat — this is a group chat
		conv.Type = "group"
		conv.Name = chatName
		if photoURL.Valid {
			conv.PhotoURL = photoURL.String
		}
	}

	// Fetch the last message preview
	lastMsg, err := db.getLastMessage(chatID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return conv, err
	}
	conv.LastMessage = lastMsg // remains nil if the chat has no messages

	// Count unread messages for this user
	unread, err := db.getUnreadCount(chatID, username)
	if err != nil {
		return conv, err
	}
	conv.UnreadCount = unread

	return conv, nil
}

func (db *appdbimpl) markMessagesReceived(username string) error {
	_, err := db.c.Exec(`
        UPDATE messageStatus
        SET status = 'received'
        WHERE status = 'sent'
          AND userOfMessageStatus = ?
    `, username)
	return err
}
