package service

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"

	"easy-im/backend/internal/apperr"
	"easy-im/backend/internal/domain"
	"easy-im/backend/internal/hub"
)

// ConversationStore persists conversations.
type ConversationStore interface {
	Create(ctx context.Context, c domain.Conversation, memberIDs []string) error
	ListForUser(ctx context.Context, userID string) ([]domain.Conversation, error)
	GetIfMember(ctx context.Context, conversationID, userID string) (domain.Conversation, error)
	MarkRead(ctx context.Context, conversationID, userID string, seq int64) (int64, error)
	FindDirectBetween(ctx context.Context, userID1, userID2 string) (domain.Conversation, error)
	ListMemberIDs(ctx context.Context, conversationID string) ([]string, error)
}

// ConversationUserLookup resolves users for open-DM.
type ConversationUserLookup interface {
	FindByID(ctx context.Context, id string) (domain.User, error)
}

// FriendshipChecker reports whether two users are accepted friends.
type FriendshipChecker interface {
	AreFriends(ctx context.Context, userID1, userID2 string) (bool, error)
}

// ConversationService implements open-DM / list / get with membership ACL.
type ConversationService struct {
	convs   ConversationStore
	users   ConversationUserLookup
	friends FriendshipChecker
	rt      RealtimePublisher // optional; nil-safe
	now     func() time.Time
}

func NewConversationService(convs ConversationStore, users ConversationUserLookup, friends FriendshipChecker, rt RealtimePublisher) *ConversationService {
	return &ConversationService{convs: convs, users: users, friends: friends, rt: rt, now: time.Now}
}

// OpenDirect get-or-creates the unique 1:1 conversation between self and peer.
// Requires an accepted friendship. Does not gate historical non-friend conversations elsewhere.
func (s *ConversationService) OpenDirect(ctx context.Context, selfUserID, peerUserID string) (domain.Conversation, error) {
	if selfUserID == "" {
		return domain.Conversation{}, apperr.Unauthorized("missing credentials")
	}
	if peerUserID == "" {
		return domain.Conversation{}, apperr.Invalid("peer user id required")
	}
	if peerUserID == selfUserID {
		return domain.Conversation{}, apperr.Invalid("cannot open conversation with yourself")
	}
	if s.convs == nil || s.users == nil || s.friends == nil {
		return domain.Conversation{}, apperr.Unavailable("database not configured")
	}

	if _, err := s.users.FindByID(ctx, peerUserID); err != nil {
		if errors.Is(err, apperr.ErrNotFound) {
			return domain.Conversation{}, apperr.NotFound("user not found")
		}
		return domain.Conversation{}, err
	}

	ok, err := s.friends.AreFriends(ctx, selfUserID, peerUserID)
	if err != nil {
		return domain.Conversation{}, err
	}
	if !ok {
		return domain.Conversation{}, apperr.Forbidden("not friends")
	}

	existing, err := s.convs.FindDirectBetween(ctx, selfUserID, peerUserID)
	if err == nil {
		return s.convs.GetIfMember(ctx, existing.ID, selfUserID)
	}
	if !errors.Is(err, apperr.ErrNotFound) {
		return domain.Conversation{}, err
	}

	now := s.now().UTC()
	c := domain.Conversation{
		ID:        uuid.NewString(),
		Title:     nil,
		CreatedBy: selfUserID,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.convs.Create(ctx, c, []string{selfUserID, peerUserID}); err != nil {
		return domain.Conversation{}, err
	}
	return s.convs.GetIfMember(ctx, c.ID, selfUserID)
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
	s.broadcastRead(ctx, conversationID, userID, last)
	return MarkReadResult{LastReadSeq: last, UnreadCount: 0}, nil
}

func (s *ConversationService) broadcastRead(ctx context.Context, conversationID, readerID string, lastReadSeq int64) {
	if s.rt == nil {
		return
	}
	memberIDs, err := s.convs.ListMemberIDs(ctx, conversationID)
	if err != nil || len(memberIDs) == 0 {
		return
	}
	payload, err := json.Marshal(map[string]any{
		"conversation_id": conversationID,
		"reader_id":       readerID,
		"last_read_seq":   lastReadSeq,
	})
	if err != nil {
		return
	}
	s.rt.PublishToUsers(memberIDs, hub.Event{Type: "message.read", Payload: payload})
}
