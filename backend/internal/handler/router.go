package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"easy-im/backend/internal/apperr"
)

// Deps are optional process dependencies for HTTP handlers.
type Deps struct {
	// Pool may be nil when DATABASE_URL is unset.
	Pool *pgxpool.Pool
	Log  *slog.Logger
}

// NewMux registers HTTP routes and standard middleware for the API process.
func NewMux(deps Deps) http.Handler {
	if deps.Log == nil {
		deps.Log = slog.New(slog.NewJSONHandler(os.Stdout, nil))
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", Healthz)
	mux.Handle("/readyz", Readyz(deps.Pool))

	var h http.Handler = mux
	h = withCORS(h)
	h = Recover(deps.Log, h)
	h = RequestID(h)
	return h
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Request-ID")
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
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(HealthResponse{Status: "ok"})
	})
}
