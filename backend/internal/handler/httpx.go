package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"easy-im/backend/internal/apperr"
)

type ctxKey int

const requestIDKey ctxKey = 1

const headerRequestID = "X-Request-ID"

// RequestIDFromContext returns the request id if present.
func RequestIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}

// withRequestID stores id on the context.
func withRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

func newRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return hex.EncodeToString([]byte("fallback-request-id"))
	}
	return hex.EncodeToString(b[:])
}

// RequestID middleware ensures every request has an X-Request-ID.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get(headerRequestID)
		if id == "" {
			id = newRequestID()
		}
		w.Header().Set(headerRequestID, id)
		next.ServeHTTP(w, r.WithContext(withRequestID(r.Context(), id)))
	})
}

// Recover catches panics and returns a stable internal error JSON.
func Recover(log *slog.Logger, next http.Handler) http.Handler {
	if log == nil {
		log = slog.Default()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Error("panic", "request_id", RequestIDFromContext(r.Context()), "panic", rec)
				WriteError(w, r, apperr.Internal("internal server error", errors.New("panic")))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

type errorBody struct {
	Error errorDetail `json:"error"`
}

type errorDetail struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id"`
}

// HTTPStatus maps an error to an HTTP status code.
func HTTPStatus(err error) int {
	ae, ok := apperr.AsApp(err)
	if !ok {
		return http.StatusInternalServerError
	}
	switch {
	case errors.Is(ae, apperr.ErrInvalid):
		return http.StatusBadRequest
	case errors.Is(ae, apperr.ErrUnauthorized):
		return http.StatusUnauthorized
	case errors.Is(ae, apperr.ErrForbidden):
		return http.StatusForbidden
	case errors.Is(ae, apperr.ErrNotFound):
		return http.StatusNotFound
	case errors.Is(ae, apperr.ErrConflict):
		return http.StatusConflict
	case errors.Is(ae, apperr.ErrUnavailable):
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

// WriteError writes a unified error JSON body. Internal details are not leaked.
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	rid := RequestIDFromContext(r.Context())
	if rid == "" {
		rid = w.Header().Get(headerRequestID)
	}

	code := "internal"
	message := "internal server error"
	status := http.StatusInternalServerError

	if ae, ok := apperr.AsApp(err); ok {
		code = ae.Code
		if ae.Message != "" {
			message = ae.Message
		}
		status = HTTPStatus(err)
		// Sanitize true internal failures only (not 503 unavailable, etc.).
		if errors.Is(ae, apperr.ErrInternal) || code == "internal" {
			message = "internal server error"
			code = "internal"
			status = http.StatusInternalServerError
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorBody{
		Error: errorDetail{
			Code:      code,
			Message:   message,
			RequestID: rid,
		},
	})
}
