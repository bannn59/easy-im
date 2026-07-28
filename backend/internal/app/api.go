package app

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"easy-im/backend/internal/handler"
	"easy-im/backend/internal/repo"
	"easy-im/backend/internal/service"
)

// APIOptions configures the API HTTP stack.
type APIOptions struct {
	Pool          *pgxpool.Pool
	Log           *slog.Logger
	AuthJWTSecret string
	AuthTokenTTL  time.Duration
}

// NewAPIHandler wires HTTP handlers for cmd/api.
func NewAPIHandler(opts APIOptions) http.Handler {
	var auth *service.AuthService
	var conv *service.ConversationService
	if opts.Pool != nil && opts.AuthJWTSecret != "" {
		users := repo.NewUserRepo(opts.Pool)
		auth = service.NewAuthService(
			users,
			service.AuthConfig{
				JWTSecret: []byte(opts.AuthJWTSecret),
				TokenTTL:  opts.AuthTokenTTL,
			},
		)
		conv = service.NewConversationService(repo.NewConversationRepo(opts.Pool), users)
	}
	return handler.NewMux(handler.Deps{
		Pool: opts.Pool,
		Log:  opts.Log,
		Auth: auth,
		Conv: conv,
	})
}
