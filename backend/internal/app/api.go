package app

import (
	"log/slog"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"easy-im/backend/internal/handler"
)

// APIOptions configures the API HTTP stack.
type APIOptions struct {
	Pool *pgxpool.Pool
	Log  *slog.Logger
}

// NewAPIHandler wires HTTP handlers for cmd/api.
func NewAPIHandler(opts APIOptions) http.Handler {
	return handler.NewMux(handler.Deps{Pool: opts.Pool, Log: opts.Log})
}
