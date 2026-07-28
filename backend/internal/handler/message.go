package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"easy-im/backend/internal/apperr"
	"easy-im/backend/internal/service"
)

// MessageHandler serves message routes under conversations.
type MessageHandler struct {
	Msg *service.MessageService
}

type sendMessageBody struct {
	Body             string `json:"body"`
	ClientMsgID      string `json:"client_msg_id"`
	ReplyToMessageID string `json:"reply_to_message_id"`
}

type replyToDTO struct {
	ID       string `json:"id"`
	SenderID string `json:"sender_id"`
	Body     string `json:"body"`
}

type messageDTO struct {
	ID             string      `json:"id"`
	ConversationID string      `json:"conversation_id"`
	SenderID       string      `json:"sender_id"`
	Body           string      `json:"body"`
	ClientMsgID    string      `json:"client_msg_id"`
	Seq            int64       `json:"seq"`
	CreatedAt      string      `json:"created_at"`
	ReplyTo        *replyToDTO `json:"reply_to"`
}

func toMessageDTO(v service.MessageView) messageDTO {
	m := v.Message
	dto := messageDTO{
		ID:             m.ID,
		ConversationID: m.ConversationID,
		SenderID:       m.SenderID,
		Body:           m.Body,
		ClientMsgID:    m.ClientMsgID,
		Seq:            m.Seq,
		CreatedAt:      m.CreatedAt.UTC().Format(timeRFC3339),
		ReplyTo:        nil,
	}
	if v.ReplyTo != nil {
		dto.ReplyTo = &replyToDTO{
			ID:       v.ReplyTo.ID,
			SenderID: v.ReplyTo.SenderID,
			Body:      v.ReplyTo.Body,
		}
	}
	return dto
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
	v, err := h.Msg.Send(r.Context(), service.SendMessageInput{
		ConversationID:   conversationID,
		SenderID:         UserIDFromContext(r.Context()),
		Body:             body.Body,
		ClientMsgID:      body.ClientMsgID,
		ReplyToMessageID: body.ReplyToMessageID,
	})
	if err != nil {
		WriteError(w, r, err)
		return
	}
	writeJSON(w, http.StatusCreated, toMessageDTO(v))
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
	for _, v := range list {
		out = append(out, toMessageDTO(v))
	}
	writeJSON(w, http.StatusOK, map[string]any{"messages": out})
}
