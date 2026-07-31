package handler

import (
	"encoding/json"
	"net/http"

	"easy-im/backend/internal/apperr"
	"easy-im/backend/internal/service"
)

// PushHandler serves /v1/push*.
type PushHandler struct {
	Push           *service.PushService
	VAPIDPublicKey string
}

type pushSubscriptionBody struct {
	Endpoint string `json:"endpoint"`
	P256DH   string `json:"p256dh"`
	Auth     string `json:"auth"`
}

// VAPID returns the public key for the browser to subscribe with.
func (h *PushHandler) VAPID(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"public_key": h.VAPIDPublicKey})
}

// Register stores the caller's Web Push subscription (cookie-authenticated).
func (h *PushHandler) Register(w http.ResponseWriter, r *http.Request) {
	if h.Push == nil {
		WriteError(w, r, apperr.Unavailable("push not configured"))
		return
	}
	var body pushSubscriptionBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, r, apperr.Invalid("invalid request body"))
		return
	}
	if err := h.Push.Register(r.Context(), UserIDFromContext(r.Context()), body.Endpoint, body.P256DH, body.Auth); err != nil {
		WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Unregister removes the caller's Web Push subscription (idempotent).
func (h *PushHandler) Unregister(w http.ResponseWriter, r *http.Request) {
	if h.Push == nil {
		WriteError(w, r, apperr.Unavailable("push not configured"))
		return
	}
	var body pushSubscriptionBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, r, apperr.Invalid("invalid request body"))
		return
	}
	if err := h.Push.Unregister(r.Context(), UserIDFromContext(r.Context()), body.Endpoint); err != nil {
		WriteError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
