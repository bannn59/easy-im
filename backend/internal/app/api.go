package app

import (
	"net/http"

	"easy-im/backend/internal/handler"
)

// NewAPIHandler wires HTTP handlers for cmd/api.
func NewAPIHandler() http.Handler {
	return handler.NewMux()
}
