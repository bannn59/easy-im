package domain

import "time"

// Conversation is a chat room (1:1 or small group).
type Conversation struct {
	ID        string
	Title     *string
	CreatedBy string
	CreatedAt time.Time
	UpdatedAt time.Time
	Members   []User // optional, filled on detail/create
}
