package handler

import (
	"log/slog"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"easy-im/backend/internal/apperr"
	"easy-im/backend/internal/hub"
	"easy-im/backend/internal/service"
)

// Deps are optional process dependencies for HTTP handlers.
type Deps struct {
	Pool    *pgxpool.Pool
	Log     *slog.Logger
	Auth    *service.AuthService
	Conv    *service.ConversationService
	Msg     *service.MessageService
	Friends *service.FriendService
	Hub     *hub.Hub
	Members service.MembershipChecker // for WS frame validation
}

// NewMux registers HTTP routes and standard middleware for the API process.
func NewMux(deps Deps) http.Handler {
	if deps.Log == nil {
		deps.Log = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", Healthz)
	mux.Handle("/readyz", Readyz(deps.Pool))

	auth := &AuthHandler{Auth: deps.Auth}
	mux.HandleFunc("/v1/auth/register", auth.Register)
	mux.HandleFunc("/v1/auth/login", auth.Login)
	require := RequireUser(deps.Auth)
	mux.Handle("GET /v1/me", require(http.HandlerFunc(auth.Me)))
	mux.Handle("PATCH /v1/me/profile", require(http.HandlerFunc(auth.UpdateProfile)))
	mux.Handle("POST /v1/me/password", require(http.HandlerFunc(auth.ChangePassword)))

	conv := &ConversationHandler{Conv: deps.Conv, Hub: deps.Hub}
	msg := &MessageHandler{Msg: deps.Msg}
	friends := &FriendHandler{Friends: deps.Friends, Conv: deps.Conv, Hub: deps.Hub}

	mux.Handle("GET /v1/conversations", require(http.HandlerFunc(conv.List)))
	mux.Handle("GET /v1/conversations/{id}", require(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		r2 := r.Clone(r.Context())
		r2.URL.Path = "/v1/conversations/" + id
		conv.Get(w, r2)
	})))
	mux.Handle("POST /v1/conversations/{id}/messages", require(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		msg.Send(w, r, r.PathValue("id"))
	})))
	mux.Handle("GET /v1/conversations/{id}/messages", require(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		msg.List(w, r, r.PathValue("id"))
	})))
	mux.Handle("POST /v1/conversations/{id}/read", require(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conv.MarkRead(w, r, r.PathValue("id"))
	})))
	mux.Handle("PATCH /v1/conversations/{id}/messages/{messageID}", require(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		msg.Edit(w, r, r.PathValue("id"), r.PathValue("messageID"))
	})))
	mux.Handle("POST /v1/conversations/{id}/messages/{messageID}/recall", require(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		msg.Recall(w, r, r.PathValue("id"), r.PathValue("messageID"))
	})))

	mux.Handle("POST /v1/friends/requests", require(http.HandlerFunc(friends.SendRequest)))
	mux.Handle("GET /v1/friends/requests/incoming", require(http.HandlerFunc(friends.ListIncoming)))
	mux.Handle("POST /v1/friends/requests/{id}/accept", require(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		friends.Accept(w, r, r.PathValue("id"))
	})))
	mux.Handle("POST /v1/friends/requests/{id}/reject", require(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		friends.Reject(w, r, r.PathValue("id"))
	})))
	mux.Handle("GET /v1/friends", require(http.HandlerFunc(friends.ListFriends)))
	mux.Handle("POST /v1/friends/{userID}/conversation", require(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		friends.OpenConversation(w, r, r.PathValue("userID"))
	})))

	ws := &WSHandler{Auth: deps.Auth, Hub: deps.Hub, Members: deps.Members, Friends: deps.Friends, Log: deps.Log}
	deps.Hub.FrameHandler = ws.HandleFrame
	deps.Hub.PresenceBroadcaster = ws.broadcastPresence
	mux.Handle("/v1/ws", ws)

	var h http.Handler = mux
	h = withCORS(h)
	h = Recover(deps.Log, h)
	h = RequestID(h)
	return h
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Request-ID, Authorization")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// Readyz reports dependency readiness (Postgres ping when configured).
func Readyz(pool *pgxpool.Pool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if pool == nil {
			WriteError(w, r, apperr.Unavailable("database not configured"))
			return
		}
		if err := pool.Ping(r.Context()); err != nil {
			WriteError(w, r, apperr.Unavailable("database unavailable"))
			return
		}
		writeJSON(w, http.StatusOK, HealthResponse{Status: "ok"})
	})
}
