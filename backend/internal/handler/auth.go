package handler

import (
	"encoding/json"
	"net/http"

	"easy-im/backend/internal/apperr"
	"easy-im/backend/internal/domain"
	"easy-im/backend/internal/service"
)

type authCredentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type publicUser struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
}

// profileDTO is /v1/me — adds member-since timestamp.
type profileDTO struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	CreatedAt   string `json:"created_at"`
}

type tokenResponse struct {
	TokenType string     `json:"token_type"`
	User      publicUser `json:"user"`
}

func toPublicUser(u domain.User) publicUser {
	return publicUser{ID: u.ID, Email: u.Email, DisplayName: u.DisplayName}
}

func toProfileDTO(u domain.User) profileDTO {
	return profileDTO{
		ID:          u.ID,
		Email:       u.Email,
		DisplayName: u.DisplayName,
		CreatedAt:   u.CreatedAt.UTC().Format(timeRFC3339),
	}
}

// AuthHandler serves /v1/auth/* and /v1/me.
type AuthHandler struct {
	Auth   *service.AuthService
	Cookie CookieConfig
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
	setSessionCookie(w, res.AccessToken, h.Cookie)
	writeJSON(w, http.StatusCreated, tokenResponse{
		TokenType: "Bearer",
		User:      toPublicUser(res.User),
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
	setSessionCookie(w, res.AccessToken, h.Cookie)
	writeJSON(w, http.StatusOK, tokenResponse{
		TokenType: "Bearer",
		User:      toPublicUser(res.User),
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	clearSessionCookie(w, h.Cookie)
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
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
	uid, err := h.Auth.ParseAccessToken(sessionToken(r))
	if err != nil {
		WriteError(w, r, err)
		return
	}
	u, err := h.Auth.Me(r.Context(), uid)
	if err != nil {
		WriteError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toProfileDTO(u))
}

type updateProfileBody struct {
	DisplayName string `json:"display_name"`
}

func (h *AuthHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.Auth == nil {
		WriteError(w, r, apperr.Unavailable("auth not configured"))
		return
	}
	var body updateProfileBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, r, apperr.Invalid("invalid JSON body"))
		return
	}
	u, err := h.Auth.UpdateDisplayName(r.Context(), UserIDFromContext(r.Context()), body.DisplayName)
	if err != nil {
		WriteError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toProfileDTO(u))
}

type changePasswordBody struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if h.Auth == nil {
		WriteError(w, r, apperr.Unavailable("auth not configured"))
		return
	}
	var body changePasswordBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, r, apperr.Invalid("invalid JSON body"))
		return
	}
	if err := h.Auth.ChangePassword(r.Context(), UserIDFromContext(r.Context()), body.CurrentPassword, body.NewPassword); err != nil {
		WriteError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
