package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"easy-im/backend/internal/apperr"
	"easy-im/backend/internal/domain"
	"easy-im/backend/internal/hub"
)

// MessageStore persists messages.
type MessageStore interface {
	Insert(ctx context.Context, m domain.Message) (domain.Message, error)
	List(ctx context.Context, conversationID string, beforeSeq int64, limit int) ([]domain.Message, error)
	FindByID(ctx context.Context, id string) (domain.Message, error)
	FindByIDs(ctx context.Context, ids []string) (map[string]domain.Message, error)
}

// MembershipChecker verifies conversation membership.
type MembershipChecker interface {
	IsMember(ctx context.Context, conversationID, userID string) (bool, error)
	ListMemberIDs(ctx context.Context, conversationID string) ([]string, error)
}

// RealtimePublisher pushes events to online users (optional).
type RealtimePublisher interface {
	PublishToUsers(userIDs []string, event hub.Event)
}

// MessageService sends and lists messages over HTTP.
type MessageService struct {
	messages MessageStore
	members  MembershipChecker
	rt       RealtimePublisher
	now      func() time.Time
}

func NewMessageService(messages MessageStore, members MembershipChecker, rt RealtimePublisher) *MessageService {
	return &MessageService{messages: messages, members: members, rt: rt, now: time.Now}
}

type SendMessageInput struct {
	ConversationID   string
	SenderID         string
	Body             string
	ClientMsgID      string
	ReplyToMessageID string // optional; empty = no reply
}

// ReplyPreview is a truncated quote shown with a message.
type ReplyPreview struct {
	ID       string
	SenderID string
	Body     string
}

// MessageView is a message plus optional reply preview for API/WS.
type MessageView struct {
	Message domain.Message
	ReplyTo *ReplyPreview
}

const replyPreviewMaxRunes = 120

func (s *MessageService) requireMember(ctx context.Context, conversationID, userID string) error {
	if userID == "" {
		return apperr.Unauthorized("missing credentials")
	}
	if s.members == nil || s.messages == nil {
		return apperr.Unavailable("database not configured")
	}
	ok, err := s.members.IsMember(ctx, conversationID, userID)
	if err != nil {
		return err
	}
	if !ok {
		return apperr.NotFound("conversation not found")
	}
	return nil
}

func truncateRunes(s string, max int) string {
	if max <= 0 || utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	return string(runes[:max])
}

func previewFrom(m domain.Message) *ReplyPreview {
	return &ReplyPreview{
		ID:       m.ID,
		SenderID: m.SenderID,
		Body:     truncateRunes(m.Body, replyPreviewMaxRunes),
	}
}

func (s *MessageService) resolveReply(ctx context.Context, conversationID, replyID string) (*string, *ReplyPreview, error) {
	replyID = strings.TrimSpace(replyID)
	if replyID == "" {
		return nil, nil, nil
	}
	target, err := s.messages.FindByID(ctx, replyID)
	if err != nil {
		if errors.Is(err, apperr.ErrNotFound) {
			return nil, nil, apperr.Invalid("reply target not found")
		}
		return nil, nil, err
	}
	if target.ConversationID != conversationID {
		return nil, nil, apperr.Invalid("reply target not in conversation")
	}
	id := target.ID
	return &id, previewFrom(target), nil
}

func (s *MessageService) hydrateViews(ctx context.Context, list []domain.Message) ([]MessageView, error) {
	ids := make([]string, 0)
	seen := map[string]struct{}{}
	for _, m := range list {
		if m.ReplyToMessageID == nil || *m.ReplyToMessageID == "" {
			continue
		}
		id := *m.ReplyToMessageID
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	var byID map[string]domain.Message
	if len(ids) > 0 {
		var err error
		byID, err = s.messages.FindByIDs(ctx, ids)
		if err != nil {
			return nil, err
		}
	}
	out := make([]MessageView, 0, len(list))
	for _, m := range list {
		v := MessageView{Message: m}
		if m.ReplyToMessageID != nil && *m.ReplyToMessageID != "" {
			if target, ok := byID[*m.ReplyToMessageID]; ok {
				v.ReplyTo = previewFrom(target)
			}
		}
		out = append(out, v)
	}
	return out, nil
}

func (s *MessageService) viewOf(ctx context.Context, m domain.Message) (MessageView, error) {
	views, err := s.hydrateViews(ctx, []domain.Message{m})
	if err != nil {
		return MessageView{}, err
	}
	if len(views) == 0 {
		return MessageView{Message: m}, nil
	}
	return views[0], nil
}

func (s *MessageService) Send(ctx context.Context, in SendMessageInput) (MessageView, error) {
	if err := s.requireMember(ctx, in.ConversationID, in.SenderID); err != nil {
		return MessageView{}, err
	}
	body := strings.TrimSpace(in.Body)
	if body == "" {
		return MessageView{}, apperr.Invalid("body is required")
	}
	if utf8.RuneCountInString(body) > 4000 {
		return MessageView{}, apperr.Invalid("body too long")
	}
	clientID := strings.TrimSpace(in.ClientMsgID)
	if clientID == "" {
		return MessageView{}, apperr.Invalid("client_msg_id is required")
	}
	if len(clientID) > 128 {
		return MessageView{}, apperr.Invalid("client_msg_id too long")
	}

	replyID, replyPreview, err := s.resolveReply(ctx, in.ConversationID, in.ReplyToMessageID)
	if err != nil {
		return MessageView{}, err
	}

	m := domain.Message{
		ID:               uuid.NewString(),
		ConversationID:   in.ConversationID,
		SenderID:         in.SenderID,
		Body:             body,
		ClientMsgID:      clientID,
		CreatedAt:        s.now().UTC(),
		ReplyToMessageID: replyID,
	}
	out, err := s.messages.Insert(ctx, m)
	if err != nil {
		return MessageView{}, err
	}
	view := MessageView{Message: out, ReplyTo: replyPreview}
	// Idempotent hit may have different stored reply; re-hydrate from store.
	if out.ID != m.ID || (out.ReplyToMessageID == nil) != (replyID == nil) {
		view, err = s.viewOf(ctx, out)
		if err != nil {
			return MessageView{}, err
		}
	}
	s.broadcast(ctx, view)
	return view, nil
}

func (s *MessageService) broadcast(ctx context.Context, v MessageView) {
	if s.rt == nil || s.members == nil {
		return
	}
	ids, err := s.members.ListMemberIDs(ctx, v.Message.ConversationID)
	if err != nil || len(ids) == 0 {
		return
	}
	payload, err := json.Marshal(messagePayload(v))
	if err != nil {
		return
	}
	s.rt.PublishToUsers(ids, hub.Event{Type: "message.created", Payload: payload})
}

func messagePayload(v MessageView) map[string]any {
	m := v.Message
	p := map[string]any{
		"id":              m.ID,
		"conversation_id": m.ConversationID,
		"sender_id":       m.SenderID,
		"body":            m.Body,
		"client_msg_id":   m.ClientMsgID,
		"seq":             m.Seq,
		"created_at":      m.CreatedAt.UTC().Format(time.RFC3339),
		"reply_to":        nil,
	}
	if v.ReplyTo != nil {
		p["reply_to"] = map[string]any{
			"id":        v.ReplyTo.ID,
			"sender_id": v.ReplyTo.SenderID,
			"body":      v.ReplyTo.Body,
		}
	}
	return p
}

func (s *MessageService) List(ctx context.Context, conversationID, userID string, beforeSeq int64, limit int) ([]MessageView, error) {
	if err := s.requireMember(ctx, conversationID, userID); err != nil {
		return nil, err
	}
	list, err := s.messages.List(ctx, conversationID, beforeSeq, limit)
	if err != nil {
		return nil, err
	}
	return s.hydrateViews(ctx, list)
}
