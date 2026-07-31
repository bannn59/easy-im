package handler

import (
	"encoding/json"
	"net/http"

	"easy-im/backend/internal/apperr"
	"easy-im/backend/internal/domain"
	"easy-im/backend/internal/hub"
	"easy-im/backend/internal/service"
)

// FriendHandler serves /v1/friends*.
type FriendHandler struct {
	Friends *service.FriendService
	Conv    *service.ConversationService
	Hub     *hub.Hub
}

type sendFriendRequestBody struct {
	Email string `json:"email"`
}

// friendWithStatusDTO is the friend list row with live online state.
type friendWithStatusDTO struct {
	ID     string `json:"id"`
	Email  string `json:"email"`
	Online bool   `json:"online"`
}

type friendRequestDTO struct {
	ID          string      `json:"id"`
	FromUserID  string      `json:"from_user_id"`
	ToUserID    string      `json:"to_user_id"`
	Status      string      `json:"status"`
	CreatedAt   string      `json:"created_at"`
	RespondedAt *string     `json:"responded_at"`
	FromUser    *publicUser `json:"from_user,omitempty"`
	ToUser      *publicUser `json:"to_user,omitempty"`
}

func toFriendRequestDTO(r domain.FriendRequest) friendRequestDTO {
	dto := friendRequestDTO{
		ID:         r.ID,
		FromUserID: r.FromUserID,
		ToUserID:   r.ToUserID,
		Status:     string(r.Status),
		CreatedAt:  r.CreatedAt.UTC().Format(timeRFC3339),
	}
	if r.RespondedAt != nil {
		s := r.RespondedAt.UTC().Format(timeRFC3339)
		dto.RespondedAt = &s
	}
	if r.FromUser != nil {
		u := toPublicUser(*r.FromUser)
		dto.FromUser = &u
	}
	if r.ToUser != nil {
		u := toPublicUser(*r.ToUser)
		dto.ToUser = &u
	}
	return dto
}

func (h *FriendHandler) SendRequest(w http.ResponseWriter, r *http.Request) {
	if h.Friends == nil {
		WriteError(w, r, apperr.Unavailable("friends not configured"))
		return
	}
	var body sendFriendRequestBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, r, apperr.Invalid("invalid JSON body"))
		return
	}
	req, err := h.Friends.SendRequest(r.Context(), UserIDFromContext(r.Context()), body.Email)
	if err != nil {
		WriteError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toFriendRequestDTO(req))
}

func (h *FriendHandler) ListIncoming(w http.ResponseWriter, r *http.Request) {
	if h.Friends == nil {
		WriteError(w, r, apperr.Unavailable("friends not configured"))
		return
	}
	list, err := h.Friends.ListIncoming(r.Context(), UserIDFromContext(r.Context()))
	if err != nil {
		WriteError(w, r, err)
		return
	}
	out := make([]friendRequestDTO, 0, len(list))
	for _, item := range list {
		out = append(out, toFriendRequestDTO(item))
	}
	writeJSON(w, http.StatusOK, map[string]any{"requests": out})
}

func (h *FriendHandler) ListFriends(w http.ResponseWriter, r *http.Request) {
	if h.Friends == nil {
		WriteError(w, r, apperr.Unavailable("friends not configured"))
		return
	}
	list, err := h.Friends.ListFriends(r.Context(), UserIDFromContext(r.Context()))
	if err != nil {
		WriteError(w, r, err)
		return
	}
	out := make([]friendWithStatusDTO, 0, len(list))
	for _, u := range list {
		online := false
		if h.Hub != nil {
			online = h.Hub.IsOnline(u.ID)
		}
		out = append(out, friendWithStatusDTO{ID: u.ID, Email: u.Email, Online: online})
	}
	writeJSON(w, http.StatusOK, map[string]any{"friends": out})
}

func (h *FriendHandler) Accept(w http.ResponseWriter, r *http.Request, requestID string) {
	if h.Friends == nil {
		WriteError(w, r, apperr.Unavailable("friends not configured"))
		return
	}
	req, err := h.Friends.Accept(r.Context(), UserIDFromContext(r.Context()), requestID)
	if err != nil {
		WriteError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toFriendRequestDTO(req))
}

func (h *FriendHandler) Reject(w http.ResponseWriter, r *http.Request, requestID string) {
	if h.Friends == nil {
		WriteError(w, r, apperr.Unavailable("friends not configured"))
		return
	}
	req, err := h.Friends.Reject(r.Context(), UserIDFromContext(r.Context()), requestID)
	if err != nil {
		WriteError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toFriendRequestDTO(req))
}

// OpenConversation get-or-creates a 1:1 conversation with an accepted friend.
func (h *FriendHandler) OpenConversation(w http.ResponseWriter, r *http.Request, peerUserID string) {
	if h.Conv == nil {
		WriteError(w, r, apperr.Unavailable("conversations not configured"))
		return
	}
	c, err := h.Conv.OpenDirect(r.Context(), UserIDFromContext(r.Context()), peerUserID)
	if err != nil {
		WriteError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toConversationDTO(c, h.Hub))
}
