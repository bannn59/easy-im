package domain

import "time"

// Message is a chat message in a conversation.
type Message struct {
	ID             string
	ConversationID string
	SenderID       string
	Body           string
	ClientMsgID    string
	Seq            int64
	CreatedAt      time.Time
}
