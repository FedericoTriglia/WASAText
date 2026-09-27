package database

import (
	"database/sql"
	"errors"
	"fmt"
)

type AppDatabase interface {
	// doLogin
	DoLogin(username string) (string, error)
	GetUserPhoto(username string) (string, error)
	CreateUser(username string) error

	// setMyPhoto
	SetUserPhoto(username string, photoURL string) error

	// getMyConversations
	GetMyConversations(username string) ([]ConversationPreview, error)
	getLastMessage(chatID int64) (*LastMessagePreview, error)
	getUnreadCount(chatID int64, username string) (int, error)
	buildConversationPreview(chatID int64, username string) (ConversationPreview, error)

	// getConversation
	GetConversation(conversationID int64, username string) (Conversation, error)
	ConversationExists(conversationID int64) (bool, error)
	IsParticipant(conversationID int64, username string) (bool, error)
	getParticipants(conversationID int64) ([]string, error)
	getMessages(conversationID int64, username string) ([]Message, error)
	getMessageStatus(messageID int64, senderUsername string, currentUsername string) (string, error)
	getReplyTo(messageID int64) (*ReplyTo, error)
	getReactions(messageID int64) ([]Reaction, error)

	// createConversation
	PrivateConversationExists(username1 string, username2 string) (bool, error)
	CreateConversation(sender string, target string, content string) (CreateConversationResult, error)

	// createGroup
	CreateGroup(name string, members []string) (CreateGroupResult, error)
	UserExists(username string) (bool, error)

	// setGroupName
	SetGroupName(groupID int64, name string) (GroupNameUpdatedResult, error)

	// setGroupPhoto
	SetGroupPhoto(groupID int64, photoURL string) (GroupPhotoUpdatedResult, error)

	// addToGroup
	GroupExists(groupID int64) (bool, error)
	AddUserToGroup(groupID int64, username string) (UsernameAddedToGroupResult, error)

	// leaveGroup
	LeaveGroup(groupID int64, username string) error

	// searchUsers
	SearchUsers(query string) ([]UserResult, error)

	// sendMessage
	InsertTextMessage(conversationID int64, sender string, content string, replyToID *int64) (MessageCreatedResult, error)
	insertMessage(conversationID int64, sender string, format string, text string, imageURL string, replyToID *int64) (MessageCreatedResult, error)
	InsertImageMessage(conversationID int64, sender string, imageURL string) (MessageCreatedResult, error)

	// deleteMessage
	IsMessageSender(messageID int64, username string) (bool, error)
	DeleteMessage(messageID int64) error

	// commentMessage
	MessageExists(messageID int64, conversationID int64) (bool, error)
	UpsertReaction(messageID int64, username string, emoticon string) (ReactionCreatedResult, error)

	// uncommentMessage
	DeleteReaction(messageID int64, username string) error

	// forwardMessage
	ForwardMessage(messageID int64, targetConversationID int64, sender string) (MessageForwardedResult, error)

	// markMessagesRead
	MarkMessagesRead(conversationID int64, username string) error
	markMessagesReceived(username string) error

	Ping() error
}

type appdbimpl struct {
	c *sql.DB
}

// New returns a new instance of AppDatabase based on the SQLite connection `db`.
// `db` is required - an error will be returned if `db` is `nil`.
func New(db *sql.DB) (AppDatabase, error) {
	if db == nil {
		return nil, errors.New("database is required when building an AppDatabase")
	}
	_, err1 := db.Exec("PRAGMA foreign_keys = ON;")
	if err1 != nil {
		return nil, fmt.Errorf("error enabling foreign keys: %w", err1)
	}
	// Check if table exists. If not, the database is empty and the structure must be created.
	var tableName string
	err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name='User';`).Scan(&tableName)
	if errors.Is(err, sql.ErrNoRows) {
		// Create tables
		sqlStmt := []string{
			`CREATE TABLE IF NOT EXISTS User (
			username TEXT PRIMARY KEY,
			photo_url TEXT
		);`,

			`CREATE TABLE IF NOT EXISTS Chat (
			id INTEGER PRIMARY KEY AUTOINCREMENT
		);`,

			`CREATE TABLE IF NOT EXISTS GroupChat (
			chat INTEGER PRIMARY KEY,
			chatName TEXT NOT NULL,
			photo_url TEXT,
		    FOREIGN KEY (chat) REFERENCES Chat(id) ON DELETE CASCADE
		);`,

			`CREATE TABLE IF NOT EXISTS PrivateChat (
			chat INTEGER PRIMARY KEY,
		    FOREIGN KEY (chat) REFERENCES Chat(id) ON DELETE CASCADE
		);`,

			`CREATE TABLE IF NOT EXISTS usChat (
			chat INTEGER NOT NULL,
			userOfChat TEXT NOT NULL,
			PRIMARY KEY (chat, userOfChat),
			FOREIGN KEY (chat) REFERENCES Chat(id) ON DELETE CASCADE,
			FOREIGN KEY (userOfChat) REFERENCES User(username) ON DELETE CASCADE ON UPDATE CASCADE
		);`,

			// CREATE INDEX IF NOT EXISTS idx_usChat_user ON usChat(user);
			// CREATE INDEX IF NOT EXISTS idx_usChat_chat ON usChat(chat);

			`CREATE TABLE IF NOT EXISTS Message (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			instant DATETIME DEFAULT CURRENT_TIMESTAMP NOT NULL,
			chat INTEGER NOT NULL,
			sender TEXT NOT NULL,
			format TEXT NOT NULL,
			text TEXT,
			image_url TEXT,
			answers INTEGER,
		    forwards INTEGER,
			FOREIGN KEY (chat) REFERENCES Chat(id) ON DELETE CASCADE,
			FOREIGN KEY (sender) REFERENCES User(username) ON DELETE CASCADE ON UPDATE CASCADE,
			FOREIGN KEY (answers) REFERENCES Message(id) ON DELETE SET NULL,
		    FOREIGN KEY (forwards) REFERENCES Message(id) ON DELETE SET NULL,
    		CHECK(format IN ('TEXT', 'IMAGE')),
    		CHECK (
				(format = 'TEXT'  AND text IS NOT NULL AND image_url IS NULL) OR
				(format = 'IMAGE' AND text IS NULL AND image_url IS NOT NULL)
    		)
		);`,

			// CREATE INDEX IF NOT EXISTS idx_Message_chat ON Message(chat);
			// CREATE INDEX IF NOT EXISTS idx_Message_sender ON Message(sender);
			// CREATE INDEX IF NOT EXISTS idx_Message_timestamp ON Message(timestamp DESC);

			`CREATE TABLE IF NOT EXISTS messageStatus (
			message INTEGER NOT NULL,
			userOfMessageStatus TEXT NOT NULL,
			status TEXT NOT NULL,
			CHECK(status IN ('sent', 'received', 'read')),
			PRIMARY KEY (message, userOfMessageStatus),
			FOREIGN KEY (message) REFERENCES Message(id) ON DELETE CASCADE,
			FOREIGN KEY (userOfMessageStatus) REFERENCES User(username) ON DELETE CASCADE ON UPDATE CASCADE
		);`,

			// CREATE INDEX IF NOT EXISTS idx_messageStatus_message ON messageStatus(message);
			// CREATE INDEX IF NOT EXISTS idx_messageStatus_user ON messageStatus(user);

			`CREATE TABLE IF NOT EXISTS reaction (
			message INTEGER NOT NULL,
			userOfReaction TEXT NOT NULL,
			emoticon TEXT NOT NULL,
			PRIMARY KEY (message, userOfReaction),
			FOREIGN KEY (message) REFERENCES Message(id) ON DELETE CASCADE,
			FOREIGN KEY (userOfReaction) REFERENCES User(username) ON DELETE CASCADE ON UPDATE CASCADE
		);`,

			// CREATE INDEX IF NOT EXISTS idx_reaction_message ON reaction(message);
			// CREATE INDEX IF NOT EXISTS idx_reaction_user ON reaction(user);
		}

		for _, stmt := range sqlStmt {
			if _, err := db.Exec(stmt); err != nil {
				return nil, fmt.Errorf("error creating database structure: %w", err)
			}
		}
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("error checking database structure: %w", err)
	}

	appdb := &appdbimpl{
		c: db,
	}
	return appdb, nil
}

func (db *appdbimpl) Ping() error {
	return db.c.Ping()
}
