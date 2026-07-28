package handler

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"easy-im/backend/internal/apperr"
	"easy-im/backend/internal/domain"
	"easy-im/backend/internal/service"
)

// ConversationHandler serves /v1/conversations.
type ConversationHandler struct {
	Conv *service.ConversationService
}

type lastMessageDTO struct {
	Seq         int64   `json:"seq"`
	Body        string  `json:"body"`
	SenderID    string  `json:"sender_id"`
	SenderEmail *string `json:"sender_email,omitempty"`
	CreatedAt   string  `json:"created_at"`
}

type conversationDTO struct {
	ID          string          `json:"id"`
	Title       *string         `json:"title"`
	CreatedBy   string          `json:"created_by"`
	CreatedAt   string          `json:"created_at"`
	UpdatedAt   string          `json:"updated_at"`
	Members     []publicUser    `json:"members,omitempty"`
	LastMessage *lastMessageDTO `json:"last_message"`
	UnreadCount int64           `json:"unread_count"`
	MemberCount int             `json:"member_count,omitempty"`
}

func toConversationDTO(c domain.Conversation) conversationDTO {
	dto := conversationDTO{
		ID:          c.ID,
		Title:       c.Title,
		CreatedBy:   c.CreatedBy,
		CreatedAt:   c.CreatedAt.UTC().Format(timeRFC3339),
		UpdatedAt:   c.UpdatedAt.UTC().Format(timeRFC3339),
		UnreadCount: c.UnreadCount,
		MemberCount: c.MemberCount,
		LastMessage: nil,
	}
	if c.LastMessageSeq != nil && c.LastMessagePreview != nil && c.LastMessageSenderID != nil && c.LastMessageAt != nil {
		dto.LastMessage = &lastMessageDTO{
			Seq:         *c.LastMessageSeq,
			Body:        *c.LastMessagePreview,
			SenderID:    *c.LastMessageSenderID,
			SenderEmail: c.LastMessageSenderEmail,
			CreatedAt:   c.LastMessageAt.UTC().Format(timeRFC3339),
		}
	}
	if len(c.Members) > 0 {
		dto.Members = make([]publicUser, 0, len(c.Members))
		for _, m := range c.Members {
			dto.Members = append(dto.Members, toPublicUser(m))
		}
	}
	return dto
}

const timeRFC3339 = "2006-01-02T15:04:05Z07:00"

func (h *ConversationHandler) List(w http.ResponseWriter, r *http.Request) {
	if h.Conv == nil {
		WriteError(w, r, apperr.Unavailable("conversations not configured"))
		return
	}
	list, err := h.Conv.List(r.Context(), UserIDFromContext(r.Context()))
	if err != nil {
		WriteError(w, r, err)
		return
	}
	out := make([]conversationDTO, 0, len(list))
	for _, c := range list {
		out = append(out, toConversationDTO(c))
	}
	writeJSON(w, http.StatusOK, map[string]any{"conversations": out})
}

func (h *ConversationHandler) Get(w http.ResponseWriter, r *http.Request) {
	if h.Conv == nil {
		WriteError(w, r, apperr.Unavailable("conversations not configured"))
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/v1/conversations/")
	id = strings.Trim(id, "/")
	if id == "" || strings.Contains(id, "/") {
		WriteError(w, r, apperr.NotFound("conversation not found"))
		return
	}
	c, err := h.Conv.Get(r.Context(), id, UserIDFromContext(r.Context()))
	if err != nil {
		WriteError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, toConversationDTO(c))
}

type markReadBody struct {
	Seq *int64 `json:"seq"`
}

func (h *ConversationHandler) MarkRead(w http.ResponseWriter, r *http.Request, conversationID string) {
	if h.Conv == nil {
		WriteError(w, r, apperr.Unavailable("conversations not configured"))
		return
	}
	var body markReadBody
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil && !errors.Is(err, io.EOF) {
			WriteError(w, r, apperr.Invalid("invalid JSON body"))
			return
		}
	}
	res, err := h.Conv.MarkRead(r.Context(), conversationID, UserIDFromContext(r.Context()), body.Seq)
	if err != nil {
		WriteError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"last_read_seq": res.LastReadSeq,
		"unread_count":  res.UnreadCount,
	})
}
