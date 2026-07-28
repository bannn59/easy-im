package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"easy-im/backend/internal/apperr"
	"easy-im/backend/internal/domain"
	"easy-im/backend/internal/service"
)

// MessageHandler serves message routes under conversations.
type MessageHandler struct {
	Msg *service.MessageService
}

type sendMessageBody struct {
	Body        string `json:"body"`
	ClientMsgID string `json:"client_msg_id"`
}

type messageDTO struct {
	ID             string `json:"id"`
	ConversationID string `json:"conversation_id"`
	SenderID       string `json:"sender_id"`
	Body           string `json:"body"`
	ClientMsgID    string `json:"client_msg_id"`
	Seq            int64  `json:"seq"`
	CreatedAt      string `json:"created_at"`
}

func toMessageDTO(m domain.Message) messageDTO {
	return messageDTO{
		ID:             m.ID,
		ConversationID: m.ConversationID,
		SenderID:       m.SenderID,
		Body:           m.Body,
		ClientMsgID:    m.ClientMsgID,
		Seq:            m.Seq,
		CreatedAt:      m.CreatedAt.UTC().Format(timeRFC3339),
	}
}

func (h *MessageHandler) Send(w http.ResponseWriter, r *http.Request, conversationID string) {
	if h.Msg == nil {
		WriteError(w, r, apperr.Unavailable("messages not configured"))
		return
	}
	var body sendMessageBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		WriteError(w, r, apperr.Invalid("invalid JSON body"))
		return
	}
	m, err := h.Msg.Send(r.Context(), service.SendMessageInput{
		ConversationID: conversationID,
		SenderID:       UserIDFromContext(r.Context()),
		Body:           body.Body,
		ClientMsgID:    body.ClientMsgID,
	})
	if err != nil {
		WriteError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toMessageDTO(m))
}

func (h *MessageHandler) List(w http.ResponseWriter, r *http.Request, conversationID string) {
	if h.Msg == nil {
		WriteError(w, r, apperr.Unavailable("messages not configured"))
		return
	}
	var before int64
	if raw := r.URL.Query().Get("before_seq"); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || n < 0 {
			WriteError(w, r, apperr.Invalid("invalid before_seq"))
			return
		}
		before = n
	}
	limit := 50
	if raw := r.URL.Query().Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			WriteError(w, r, apperr.Invalid("invalid limit"))
			return
		}
		limit = n
	}
	list, err := h.Msg.List(r.Context(), conversationID, UserIDFromContext(r.Context()), before, limit)
	if err != nil {
		WriteError(w, r, err)
		return
	}
	out := make([]messageDTO, 0, len(list))
	for _, m := range list {
		out = append(out, toMessageDTO(m))
	}
	writeJSON(w, http.StatusOK, map[string]any{"messages": out})
}
