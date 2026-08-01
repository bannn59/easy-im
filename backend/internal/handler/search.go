package handler

import (
	"net/http"
	"strconv"

	"easy-im/backend/internal/apperr"
	"easy-im/backend/internal/domain"
	"easy-im/backend/internal/service"
)

// searchResultDTO is a global-search hit: message fields plus conversation
// context for display and navigation.
type searchResultDTO struct {
	ConversationID    string      `json:"conversation_id"`
	ConversationTitle *string     `json:"conversation_title"`
	ID                string      `json:"id"`
	SenderID          string      `json:"sender_id"`
	Body              string      `json:"body"`
	ClientMsgID       string      `json:"client_msg_id"`
	Seq               int64       `json:"seq"`
	CreatedAt         string      `json:"created_at"`
	ReplyTo           *replyToDTO `json:"reply_to"`
	EditedAt          *string     `json:"edited_at"`
	RecalledAt        *string     `json:"recalled_at"`
}

func toSearchResultDTO(v service.GlobalSearchResultView) searchResultDTO {
	msg := v.Message.Message
	dto := searchResultDTO{
		ConversationID:    msg.ConversationID,
		ConversationTitle: v.ConversationTitle,
		ID:                msg.ID,
		SenderID:          msg.SenderID,
		Body:              msg.Body,
		ClientMsgID:       msg.ClientMsgID,
		Seq:               msg.Seq,
		CreatedAt:         msg.CreatedAt.UTC().Format(timeRFC3339),
		ReplyTo:           nil,
		EditedAt:          nil,
		RecalledAt:        nil,
	}
	if msg.EditedAt != nil {
		s := msg.EditedAt.UTC().Format(timeRFC3339)
		dto.EditedAt = &s
	}
	if msg.RecalledAt != nil {
		s := msg.RecalledAt.UTC().Format(timeRFC3339)
		dto.RecalledAt = &s
	}
	if v.Message.ReplyTo != nil {
		dto.ReplyTo = &replyToDTO{
			ID:       v.Message.ReplyTo.ID,
			SenderID: v.Message.ReplyTo.SenderID,
			Body:     v.Message.ReplyTo.Body,
		}
	}
	return dto
}

// GlobalSearch searches messages across all conversations the authenticated
// user belongs to.
func (h *MessageHandler) GlobalSearch(w http.ResponseWriter, r *http.Request) {
	if h.Msg == nil {
		WriteError(w, r, apperr.Unavailable("messages not configured"))
		return
	}
	q := r.URL.Query()
	query := q.Get("q")
	if query == "" {
		WriteError(w, r, apperr.Invalid("q is required"))
		return
	}
	cursor, err := domain.ParseSearchCursor(q.Get("cursor"))
	if err != nil {
		WriteError(w, r, apperr.Invalid("invalid cursor"))
		return
	}
	limit := 50
	if raw := q.Get("limit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			WriteError(w, r, apperr.Invalid("invalid limit"))
			return
		}
		limit = n
	}
	results, next, err := h.Msg.GlobalSearch(r.Context(), UserIDFromContext(r.Context()), query, cursor, limit)
	if err != nil {
		WriteError(w, r, err)
		return
	}
	out := make([]searchResultDTO, 0, len(results))
	for _, v := range results {
		out = append(out, toSearchResultDTO(v))
	}
	var nextStr string
	if next != nil {
		nextStr = next.String()
	}
	writeJSON(w, http.StatusOK, map[string]any{"messages": out, "next_cursor": nextStr})
}
