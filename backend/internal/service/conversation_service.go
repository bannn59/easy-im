package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	"easy-im/backend/internal/apperr"
	"easy-im/backend/internal/domain"
)

// ConversationStore persists conversations.
type ConversationStore interface {
	Create(ctx context.Context, c domain.Conversation, memberIDs []string) error
	ListForUser(ctx context.Context, userID string) ([]domain.Conversation, error)
	GetIfMember(ctx context.Context, conversationID, userID string) (domain.Conversation, error)
	MarkRead(ctx context.Context, conversationID, userID string, seq int64) (int64, error)
}

// UserEmailLookup resolves emails to user ids.
type UserEmailLookup interface {
	FindIDsByEmails(ctx context.Context, emails []string) (map[string]string, error)
	FindByID(ctx context.Context, id string) (domain.User, error)
}

// ConversationService implements create/list/get with membership ACL.
type ConversationService struct {
	convs ConversationStore
	users UserEmailLookup
	now   func() time.Time
}

func NewConversationService(convs ConversationStore, users UserEmailLookup) *ConversationService {
	return &ConversationService{convs: convs, users: users, now: time.Now}
}

type CreateConversationInput struct {
	Title         string
	MemberEmails  []string
	CreatorUserID string
}

func (s *ConversationService) Create(ctx context.Context, in CreateConversationInput) (domain.Conversation, error) {
	if in.CreatorUserID == "" {
		return domain.Conversation{}, apperr.Unauthorized("missing credentials")
	}
	if s.convs == nil || s.users == nil {
		return domain.Conversation{}, apperr.Unavailable("database not configured")
	}

	emails := normalizeEmailList(in.MemberEmails)
	idMap, err := s.users.FindIDsByEmails(ctx, emails)
	if err != nil {
		return domain.Conversation{}, err
	}
	var missing []string
	for _, e := range emails {
		if _, ok := idMap[e]; !ok {
			missing = append(missing, e)
		}
	}
	if len(missing) > 0 {
		return domain.Conversation{}, apperr.Invalid("unknown member email: " + strings.Join(missing, ", "))
	}

	memberSet := map[string]struct{}{in.CreatorUserID: {}}
	for _, id := range idMap {
		memberSet[id] = struct{}{}
	}
	memberIDs := make([]string, 0, len(memberSet))
	for id := range memberSet {
		memberIDs = append(memberIDs, id)
	}

	now := s.now().UTC()
	var titlePtr *string
	title := strings.TrimSpace(in.Title)
	if title != "" {
		titlePtr = &title
	}
	c := domain.Conversation{
		ID:        uuid.NewString(),
		Title:     titlePtr,
		CreatedBy: in.CreatorUserID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.convs.Create(ctx, c, memberIDs); err != nil {
		return domain.Conversation{}, err
	}
	return s.convs.GetIfMember(ctx, c.ID, in.CreatorUserID)
}

func (s *ConversationService) List(ctx context.Context, userID string) ([]domain.Conversation, error) {
	if userID == "" {
		return nil, apperr.Unauthorized("missing credentials")
	}
	if s.convs == nil {
		return nil, apperr.Unavailable("database not configured")
	}
	return s.convs.ListForUser(ctx, userID)
}

func (s *ConversationService) Get(ctx context.Context, conversationID, userID string) (domain.Conversation, error) {
	if userID == "" {
		return domain.Conversation{}, apperr.Unauthorized("missing credentials")
	}
	if conversationID == "" {
		return domain.Conversation{}, apperr.Invalid("conversation id required")
	}
	if s.convs == nil {
		return domain.Conversation{}, apperr.Unavailable("database not configured")
	}
	return s.convs.GetIfMember(ctx, conversationID, userID)
}

// MarkReadResult is returned after advancing the member read cursor.
type MarkReadResult struct {
	LastReadSeq int64
	UnreadCount int64
}

// MarkRead sets last_read_seq to at least seq (default: conversation head seq).
func (s *ConversationService) MarkRead(ctx context.Context, conversationID, userID string, seq *int64) (MarkReadResult, error) {
	if userID == "" {
		return MarkReadResult{}, apperr.Unauthorized("missing credentials")
	}
	if conversationID == "" {
		return MarkReadResult{}, apperr.Invalid("conversation id required")
	}
	if s.convs == nil {
		return MarkReadResult{}, apperr.Unavailable("database not configured")
	}
	c, err := s.convs.GetIfMember(ctx, conversationID, userID)
	if err != nil {
		return MarkReadResult{}, err
	}
	target := int64(0)
	if c.LastMessageSeq != nil {
		target = *c.LastMessageSeq
	}
	if seq != nil {
		if *seq < 0 {
			return MarkReadResult{}, apperr.Invalid("invalid seq")
		}
		target = *seq
		if c.LastMessageSeq != nil && target > *c.LastMessageSeq {
			target = *c.LastMessageSeq
		}
	}
	last, err := s.convs.MarkRead(ctx, conversationID, userID, target)
	if err != nil {
		return MarkReadResult{}, err
	}
	return MarkReadResult{LastReadSeq: last, UnreadCount: 0}, nil
}

func normalizeEmailList(in []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, raw := range in {
		e := strings.TrimSpace(strings.ToLower(raw))
		if e == "" {
			continue
		}
		if _, ok := seen[e]; ok {
			continue
		}
		seen[e] = struct{}{}
		out = append(out, e)
	}
	return out
}
