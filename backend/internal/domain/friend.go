package domain

import "time"

// FriendRequestStatus is the lifecycle of a friend request.
type FriendRequestStatus string

const (
	FriendRequestPending  FriendRequestStatus = "pending"
	FriendRequestAccepted FriendRequestStatus = "accepted"
	FriendRequestRejected FriendRequestStatus = "rejected"
)

// FriendRequest is a directed request from one user to another.
type FriendRequest struct {
	ID          string
	FromUserID  string
	ToUserID    string
	Status      FriendRequestStatus
	CreatedAt   time.Time
	RespondedAt *time.Time

	// Optional hydrations for list/detail responses.
	FromUser *User
	ToUser   *User
}
