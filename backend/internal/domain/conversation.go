package domain

import "time"

// Conversation is a chat room (1:1 or small group).
type Conversation struct {
	ID        string
	Title     *string
	CreatedBy string
	CreatedAt time.Time
	UpdatedAt time.Time
	Members   []User // filled on detail/create; also list (sidebar titles)

	// Last message head (denormalized; nil/empty when never messaged).
	LastMessageAt          *time.Time
	LastMessageSeq         *int64
	LastMessagePreview     *string
	LastMessageSenderID    *string
	LastMessageSenderEmail *string // list hydrate; optional

	// List-only for the requesting member.
	LastReadSeq int64
	UnreadCount int64
	MemberCount int // list hydrate for DM vs group preview rules
}
