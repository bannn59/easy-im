package service

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"easy-im/backend/internal/apperr"
	"easy-im/backend/internal/domain"
)

// MessageStore persists messages.
type MessageStore interface {
	Insert(ctx context.Context, m domain.Message) (domain.Message, error)
	List(ctx context.Context, conversationID string, beforeSeq int64, limit int) ([]domain.Message, error)
}

// MembershipChecker verifies conversation membership.
type MembershipChecker interface {
	IsMember(ctx context.Context, conversationID, userID string) (bool, error)
}

// MessageService sends and lists messages over HTTP.
type MessageService struct {
	messages MessageStore
	members  MembershipChecker
	now      func() time.Time
}

func NewMessageService(messages MessageStore, members MembershipChecker) *MessageService {
	return &MessageService{messages: messages, members: members, now: time.Now}
}

type SendMessageInput struct {
	ConversationID string
	SenderID       string
	Body           string
	ClientMsgID    string
}

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

func (s *MessageService) Send(ctx context.Context, in SendMessageInput) (domain.Message, error) {
	if err := s.requireMember(ctx, in.ConversationID, in.SenderID); err != nil {
		return domain.Message{}, err
	}
	body := strings.TrimSpace(in.Body)
	if body == "" {
		return domain.Message{}, apperr.Invalid("body is required")
	}
	if utf8.RuneCountInString(body) > 4000 {
		return domain.Message{}, apperr.Invalid("body too long")
	}
	clientID := strings.TrimSpace(in.ClientMsgID)
	if clientID == "" {
		return domain.Message{}, apperr.Invalid("client_msg_id is required")
	}
	if len(clientID) > 128 {
		return domain.Message{}, apperr.Invalid("client_msg_id too long")
	}

	m := domain.Message{
		ID:             uuid.NewString(),
		ConversationID: in.ConversationID,
		SenderID:       in.SenderID,
		Body:           body,
		ClientMsgID:    clientID,
		CreatedAt:      s.now().UTC(),
	}
	return s.messages.Insert(ctx, m)
}

func (s *MessageService) List(ctx context.Context, conversationID, userID string, beforeSeq int64, limit int) ([]domain.Message, error) {
	if err := s.requireMember(ctx, conversationID, userID); err != nil {
		return nil, err
	}
	return s.messages.List(ctx, conversationID, beforeSeq, limit)
}
