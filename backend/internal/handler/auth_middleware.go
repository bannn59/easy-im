package handler

import (
	"context"
	"net/http"

	"easy-im/backend/internal/apperr"
	"easy-im/backend/internal/service"
)

type authUserKey struct{}

// UserIDFromContext returns the authenticated user id if present.
func UserIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(authUserKey{}).(string); ok {
		return v
	}
	return ""
}

// RequireUser middleware validates Bearer JWT and injects user id into context.
func RequireUser(auth *service.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if auth == nil {
				WriteError(w, r, apperr.Unavailable("auth not configured"))
				return
			}
			uid, err := auth.ParseAccessToken(sessionToken(r))
			if err != nil {
				WriteError(w, r, err)
				return
			}
			ctx := context.WithValue(r.Context(), authUserKey{}, uid)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
