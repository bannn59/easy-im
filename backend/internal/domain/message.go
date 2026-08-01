package domain

import (
	"errors"
	"strings"
	"time"
)

// Message is a chat message in a conversation.
type Message struct {
	ID               string
	ConversationID   string
	SenderID         string
	Body             string
	ClientMsgID      string
	Seq              int64
	CreatedAt        time.Time
	ReplyToMessageID *string
	EditedAt         *time.Time
	RecalledAt       *time.Time
}

// SearchCursor is a composite pagination cursor for cross-conversation search.
// Conversation seq is only unique within a conversation, so global search pages
// by (created_at, id). Serialized as "created_at|id" in query strings.
type SearchCursor struct {
	CreatedAt time.Time
	ID        string
}

// ParseSearchCursor decodes the "created_at|id" cursor string. Empty input
// returns a nil cursor (start from the latest).
func ParseSearchCursor(raw string) (*SearchCursor, error) {
	if raw == "" {
		return nil, nil
	}
	at := strings.LastIndexByte(raw, '|')
	if at < 0 {
		return nil, errors.New("malformed cursor")
	}
	ts, err := time.Parse(time.RFC3339Nano, raw[:at])
	if err != nil {
		return nil, err
	}
	return &SearchCursor{CreatedAt: ts, ID: raw[at+1:]}, nil
}

// String serializes the cursor for a next_cursor response.
func (c *SearchCursor) String() string {
	if c == nil {
		return ""
	}
	return c.CreatedAt.UTC().Format(time.RFC3339Nano) + "|" + c.ID
}

// GlobalSearchResult is a message plus its conversation title for global search.
type GlobalSearchResult struct {
	Message           Message
	ConversationTitle *string
}

