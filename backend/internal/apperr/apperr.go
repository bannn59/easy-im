// Package apperr defines application errors with stable machine codes.
// It must not import net/http — handlers map these to protocol responses.
package apperr

import (
	"errors"
	"fmt"
)

// Sentinel kinds for errors.Is / classification.
var (
	ErrNotFound     = errors.New("not found")
	ErrUnauthorized = errors.New("unauthorized")
	ErrForbidden    = errors.New("forbidden")
	ErrInvalid      = errors.New("invalid")
	ErrConflict     = errors.New("conflict")
	ErrUnavailable  = errors.New("unavailable")
	ErrInternal     = errors.New("internal")
)

// Error is a client-safe application error.
type Error struct {
	// Code is a stable machine code, e.g. "not_found".
	Code string
	// Message is safe to return to clients.
	Message string
	// Err is the underlying error (may be a sentinel or wrapped cause).
	Err error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Code, e.Err)
	}
	return e.Code
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

// New builds an Error with the given code, message, and optional cause.
func New(code, message string, err error) *Error {
	return &Error{Code: code, Message: message, Err: err}
}

func NotFound(message string) *Error {
	return New("not_found", message, ErrNotFound)
}

func Unauthorized(message string) *Error {
	return New("unauthorized", message, ErrUnauthorized)
}

func Forbidden(message string) *Error {
	return New("forbidden", message, ErrForbidden)
}

func Invalid(message string) *Error {
	return New("invalid_argument", message, ErrInvalid)
}

func Conflict(message string) *Error {
	return New("conflict", message, ErrConflict)
}

func Unavailable(message string) *Error {
	return New("unavailable", message, ErrUnavailable)
}

func Internal(message string, err error) *Error {
	if message == "" {
		message = "internal server error"
	}
	if err == nil {
		err = ErrInternal
	}
	return New("internal", message, err)
}

// AsApp returns *Error if err is or wraps one.
func AsApp(err error) (*Error, bool) {
	var ae *Error
	if errors.As(err, &ae) {
		return ae, true
	}
	return nil, false
}
