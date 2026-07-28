package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"easy-im/backend/internal/apperr"
	"easy-im/backend/internal/domain"
	"easy-im/backend/internal/service"
)

type authCredentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type publicUser struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

type tokenResponse struct {
	AccessToken string     `json:"access_token"`
	TokenType   string     `json:"token_type"`
	User        publicUser `json:"user"`
}

func toPublicUser(u domain.User) publicUser {
	return publicUser{ID: u.ID, Email: u.Email}
}

// AuthHandler serves /v1/auth/* and /v1/me.
type AuthHandler struct {
	Auth *service.AuthService
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body authCredentials
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, r, apperr.Invalid("invalid JSON body"))
		return
	}
	if h.Auth == nil {
		WriteError(w, r, apperr.Unavailable("auth not configured"))
		return
	}
	res, err := h.Auth.Register(r.Context(), body.Email, body.Password)
	if err != nil {
		WriteError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, tokenResponse{
		AccessToken: res.AccessToken,
		TokenType:   "Bearer",
		User:        toPublicUser(res.User),
	})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body authCredentials
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, r, apperr.Invalid("invalid JSON body"))
		return
	}
	if h.Auth == nil {
		WriteError(w, r, apperr.Unavailable("auth not configured"))
		return
	}
	res, err := h.Auth.Login(r.Context(), body.Email, body.Password)
	if err != nil {
		WriteError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, tokenResponse{
		AccessToken: res.AccessToken,
		TokenType:   "Bearer",
		User:        toPublicUser(res.User),
	})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.Auth == nil {
		WriteError(w, r, apperr.Unavailable("auth not configured"))
		return
	}
	uid, err := h.Auth.ParseAccessToken(bearerToken(r))
	if err != nil {
		WriteError(w, r, err)
		return
	}
	u, err := h.Auth.Me(r.Context(), uid)
	if err != nil {
		WriteError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toPublicUser(u))
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if h == "" {
		return ""
	}
	const p = "Bearer "
	if len(h) < len(p) || !strings.EqualFold(h[:len(p)], p) {
		return ""
	}
	return strings.TrimSpace(h[len(p):])
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
