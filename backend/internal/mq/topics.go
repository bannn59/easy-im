package mq

import (
	"time"

	"easy-im/backend/internal/domain"
)

// Kafka topic names for the event bus. Keys partition per conversation
// (messages) and per user (presence) to preserve ordering.
const (
	TopicMessages = "im.messages"
	TopicPresence = "im.presence"
)

// MessageEvent is published on TopicMessages after a message is durably stored.
// Key: conversation_id. Consumers use it to fan out offline pushes.
type MessageEvent struct {
	ID             string    `json:"id"`
	ConversationID string    `json:"conversation_id"`
	SenderID       string    `json:"sender_id"`
	Body           string    `json:"body"`
	CreatedAt      time.Time `json:"created_at"`
}

// NewMessageEvent builds a MessageEvent from a stored message.
func NewMessageEvent(m domain.Message) MessageEvent {
	return MessageEvent{
		ID:             m.ID,
		ConversationID: m.ConversationID,
		SenderID:       m.SenderID,
		Body:           m.Body,
		CreatedAt:      m.CreatedAt,
	}
}

// ToDomain maps the event back to a message for consumers.
func (e MessageEvent) ToDomain() domain.Message {
	return domain.Message{
		ID:             e.ID,
		ConversationID: e.ConversationID,
		SenderID:       e.SenderID,
		Body:           e.Body,
		CreatedAt:      e.CreatedAt,
	}
}

// PresenceEvent is published on TopicPresence on online/offline transitions.
// Key: user_id.
type PresenceEvent struct {
	UserID string    `json:"user_id"`
	Online bool      `json:"online"`
	At     time.Time `json:"at"`
}
