package app

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"easy-im/backend/internal/handler"
	"easy-im/backend/internal/hub"
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
	rtHub := hub.New()
	var auth *service.AuthService
	var conv *service.ConversationService
	var msg *service.MessageService
	var friends *service.FriendService
	if opts.Pool != nil && opts.AuthJWTSecret != "" {
		users := repo.NewUserRepo(opts.Pool)
		convs := repo.NewConversationRepo(opts.Pool)
		messages := repo.NewMessageRepo(opts.Pool)
		friendRepo := repo.NewFriendRepo(opts.Pool)
		auth = service.NewAuthService(
			users,
			service.AuthConfig{
				JWTSecret: []byte(opts.AuthJWTSecret),
				TokenTTL:  opts.AuthTokenTTL,
			},
		)
		conv = service.NewConversationService(convs, users)
		msg = service.NewMessageService(messages, convs, rtHub)
		friends = service.NewFriendService(friendRepo, users)
	}
	return handler.NewMux(handler.Deps{
		Pool:    opts.Pool,
		Log:     opts.Log,
		Auth:    auth,
		Conv:    conv,
		Msg:     msg,
		Friends: friends,
		Hub:     rtHub,
	})
}
