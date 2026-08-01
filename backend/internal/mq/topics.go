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

// MessageEventType discriminates the kind of message event on TopicMessages.
type MessageEventType string

const (
	// MessageCreated is a newly sent message (default; also the offline-push event).
	MessageCreated MessageEventType = "created"
	// MessageEdited is an own-message body edit within the edit window.
	MessageEdited MessageEventType = "edited"
	// MessageRecalled is an own-message recall within the recall window.
	MessageRecalled MessageEventType = "recalled"
	// MessageRead is a member advancing the conversation read cursor.
	MessageRead MessageEventType = "read"
	// GroupMembersChanged is a group membership/owner change (added/left/kicked/owner_transferred).
	GroupMembersChanged MessageEventType = "group.members_changed"
	// GroupConversationRenamed is a group display-title change.
	GroupConversationRenamed MessageEventType = "group.conversation_renamed"
)

// MessageEvent is published on TopicMessages after a message state change.
// Key: conversation_id. Consumers fan out realtime delivery and offline pushes.
//
// Fields are shared between the offline-push worker (created only) and the
// per-node realtime fanout consumer. New fields must be omitempty so older
// consumers decoding a "created" record keep working.
type MessageEvent struct {
	Type           string    `json:"type,omitempty"`              // MessageEventType; empty means "created" (back-compat)
	Origin         string    `json:"origin,omitempty"`            // node/process id that produced this event
	ID             string    `json:"id"`
	ConversationID string    `json:"conversation_id"`
	SenderID       string    `json:"sender_id,omitempty"`
	Body           string    `json:"body,omitempty"`
	CreatedAt      time.Time `json:"created_at,omitempty"`
	EditedAt       time.Time `json:"edited_at,omitempty"`
	RecalledAt     time.Time `json:"recalled_at,omitempty"`
	// ReadEvent fields (Type == "read").
	ReadByUserID string `json:"read_by_user_id,omitempty"`
	LastReadSeq  int64  `json:"last_read_seq,omitempty"`

	// Group-event fields (Type == "group.members_changed" or "group.conversation_renamed").
	// Action is the membership action: added | left | kicked | owner_transferred.
	Action    string    `json:"action,omitempty"`
	ActorID   string    `json:"actor_id,omitempty"`
	MemberIDs []string  `json:"member_ids,omitempty"`
	Title     string    `json:"title,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

// NewMessageEvent builds a "created" MessageEvent from a stored message.
func NewMessageEvent(m domain.Message) MessageEvent {
	return MessageEvent{
		Type:           string(MessageCreated),
		ID:             m.ID,
		ConversationID: m.ConversationID,
		SenderID:       m.SenderID,
		Body:           m.Body,
		CreatedAt:      m.CreatedAt,
	}
}

// NewEditedEvent builds an "edited" MessageEvent.
func NewEditedEvent(m domain.Message, origin string) MessageEvent {
	editedAt := time.Time{}
	if m.EditedAt != nil {
		editedAt = *m.EditedAt
	}
	return MessageEvent{
		Type:           string(MessageEdited),
		Origin:         origin,
		ID:             m.ID,
		ConversationID: m.ConversationID,
		SenderID:       m.SenderID,
		Body:           m.Body,
		CreatedAt:      m.CreatedAt,
		EditedAt:       editedAt,
	}
}

// NewRecalledEvent builds a "recalled" MessageEvent.
func NewRecalledEvent(m domain.Message, origin string) MessageEvent {
	recalledAt := time.Time{}
	if m.RecalledAt != nil {
		recalledAt = *m.RecalledAt
	}
	return MessageEvent{
		Type:           string(MessageRecalled),
		Origin:         origin,
		ID:             m.ID,
		ConversationID: m.ConversationID,
		SenderID:       m.SenderID,
		CreatedAt:      m.CreatedAt,
		RecalledAt:     recalledAt,
	}
}

// NewReadEvent builds a "read" MessageEvent.
func NewReadEvent(conversationID, userID string, lastReadSeq int64, origin string) MessageEvent {
	return MessageEvent{
		Type:           string(MessageRead),
		Origin:         origin,
		ConversationID: conversationID,
		ReadByUserID:   userID,
		LastReadSeq:    lastReadSeq,
	}
}

// NewMembersChangedEvent builds a "group.members_changed" MessageEvent carrying
// the post-change member id list so the fanout consumer can scope delivery
// without re-querying membership.
func NewMembersChangedEvent(conversationID, action, actorID string, members []string, origin string) MessageEvent {
	return MessageEvent{
		Type:           string(GroupMembersChanged),
		Origin:         origin,
		ConversationID: conversationID,
		Action:         action,
		ActorID:        actorID,
		MemberIDs:      members,
	}
}

// NewConversationRenamedEvent builds a "group.conversation_renamed" MessageEvent.
// The title travels with the event so the fanout consumer can rebuild the WS
// frame without a DB read.
func NewConversationRenamedEvent(conversationID, title string, updatedAt time.Time, origin string) MessageEvent {
	return MessageEvent{
		Type:           string(GroupConversationRenamed),
		Origin:         origin,
		ConversationID: conversationID,
		Title:          title,
		UpdatedAt:      updatedAt,
	}
}

// EventType returns the normalized event type, defaulting to "created" for
// older records that predate the type field.
func (e MessageEvent) EventType() MessageEventType {
	if e.Type == "" {
		return MessageCreated
	}
	return MessageEventType(e.Type)
}

// ToDomain maps the event back to a message for consumers.
func (e MessageEvent) ToDomain() domain.Message {
	m := domain.Message{
		ID:             e.ID,
		ConversationID: e.ConversationID,
		SenderID:       e.SenderID,
		Body:           e.Body,
		CreatedAt:      e.CreatedAt,
	}
	if !e.EditedAt.IsZero() {
		t := e.EditedAt
		m.EditedAt = &t
	}
	if !e.RecalledAt.IsZero() {
		t := e.RecalledAt
		m.RecalledAt = &t
	}
	return m
}

// PresenceEvent is published on TopicPresence on online/offline transitions.
// Key: user_id.
type PresenceEvent struct {
	UserID string    `json:"user_id"`
	Online bool      `json:"online"`
	At     time.Time `json:"at"`
}
