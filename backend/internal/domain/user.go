package domain

import "time"

// User is the public user entity (never includes password hash).
type User struct {
	ID        string
	Email     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// UserRecord is the persistence shape including password hash.
type UserRecord struct {
	User
	PasswordHash string
}
