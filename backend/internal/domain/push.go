package domain

import "time"

// PushSubscription is a browser Web Push endpoint registered by a user for
// offline notification delivery. p256dh and auth are the base64url-encoded
// client public key / auth secret from the browser PushSubscription.
type PushSubscription struct {
	ID        string
	UserID    string
	Endpoint  string
	P256DH    string
	Auth      string
	CreatedAt time.Time
	UpdatedAt time.Time
}
