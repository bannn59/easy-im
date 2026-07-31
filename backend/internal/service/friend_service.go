package service

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"

	"easy-im/backend/internal/apperr"
	"easy-im/backend/internal/domain"
)

// FriendStore persists friend requests and friendships.
type FriendStore interface {
	CreateRequest(ctx context.Context, req domain.FriendRequest) error
	GetRequestByID(ctx context.Context, id string) (domain.FriendRequest, error)
	FindPending(ctx context.Context, fromUserID, toUserID string) (domain.FriendRequest, error)
	AreFriends(ctx context.Context, userID1, userID2 string) (bool, error)
	ListIncomingPending(ctx context.Context, userID string) ([]domain.FriendRequest, error)
	ListFriends(ctx context.Context, userID string) ([]domain.User, error)
	ListFriendIDs(ctx context.Context, userID string) ([]string, error)
	AcceptRequest(ctx context.Context, requestID, actorUserID string, respondedAt time.Time) (domain.FriendRequest, error)
	RejectRequest(ctx context.Context, requestID, actorUserID string, respondedAt time.Time) (domain.FriendRequest, error)
}

// FriendUserLookup resolves users for friend flows.
type FriendUserLookup interface {
	FindByEmail(ctx context.Context, email string) (domain.UserRecord, error)
	FindByID(ctx context.Context, id string) (domain.User, error)
}

// FriendService implements request / accept / reject / list use-cases.
type FriendService struct {
	friends FriendStore
	users   FriendUserLookup
	now     func() time.Time
}

func NewFriendService(friends FriendStore, users FriendUserLookup) *FriendService {
	return &FriendService{friends: friends, users: users, now: time.Now}
}

func (s *FriendService) ensureReady() error {
	if s.friends == nil || s.users == nil {
		return apperr.Unavailable("database not configured")
	}
	return nil
}

// SendRequest creates a pending friend request to the user identified by peerEmail.
func (s *FriendService) SendRequest(ctx context.Context, fromUserID, peerEmail string) (domain.FriendRequest, error) {
	if fromUserID == "" {
		return domain.FriendRequest{}, apperr.Unauthorized("missing credentials")
	}
	if err := s.ensureReady(); err != nil {
		return domain.FriendRequest{}, err
	}

	email, err := normalizeFriendEmail(peerEmail)
	if err != nil {
		return domain.FriendRequest{}, err
	}

	peer, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, apperr.ErrNotFound) {
			// Stable client-safe failure for unknown emails (AC6).
			return domain.FriendRequest{}, apperr.Invalid("user not found")
		}
		return domain.FriendRequest{}, err
	}
	if peer.ID == fromUserID {
		return domain.FriendRequest{}, apperr.Invalid("cannot send friend request to yourself")
	}

	ok, err := s.friends.AreFriends(ctx, fromUserID, peer.ID)
	if err != nil {
		return domain.FriendRequest{}, err
	}
	if ok {
		return domain.FriendRequest{}, apperr.Conflict("already friends")
	}

	if _, err := s.friends.FindPending(ctx, fromUserID, peer.ID); err == nil {
		return domain.FriendRequest{}, apperr.Conflict("friend request already pending")
	} else if !errors.Is(err, apperr.ErrNotFound) {
		return domain.FriendRequest{}, err
	}
	// Block dual directed pending for the same pair (either side already asked).
	if _, err := s.friends.FindPending(ctx, peer.ID, fromUserID); err == nil {
		return domain.FriendRequest{}, apperr.Conflict("friend request already pending")
	} else if !errors.Is(err, apperr.ErrNotFound) {
		return domain.FriendRequest{}, err
	}

	now := s.now().UTC()
	req := domain.FriendRequest{
		ID:         uuid.NewString(),
		FromUserID: fromUserID,
		ToUserID:   peer.ID,
		Status:     domain.FriendRequestPending,
		CreatedAt:  now,
	}
	if err := s.friends.CreateRequest(ctx, req); err != nil {
		return domain.FriendRequest{}, err
	}

	fromUser, err := s.users.FindByID(ctx, fromUserID)
	if err != nil {
		// Request is created; return bare request if hydrate fails.
		return req, nil
	}
	toUser := peer.User
	req.FromUser = &fromUser
	req.ToUser = &toUser
	return req, nil
}

func (s *FriendService) ListIncoming(ctx context.Context, userID string) ([]domain.FriendRequest, error) {
	if userID == "" {
		return nil, apperr.Unauthorized("missing credentials")
	}
	if err := s.ensureReady(); err != nil {
		return nil, err
	}
	return s.friends.ListIncomingPending(ctx, userID)
}

func (s *FriendService) ListFriends(ctx context.Context, userID string) ([]domain.User, error) {
	if userID == "" {
		return nil, apperr.Unauthorized("missing credentials")
	}
	if err := s.ensureReady(); err != nil {
		return nil, err
	}
	return s.friends.ListFriends(ctx, userID)
}

// ListFriendIDs returns the user IDs of all accepted friends.
func (s *FriendService) ListFriendIDs(ctx context.Context, userID string) ([]string, error) {
	if userID == "" {
		return nil, apperr.Unauthorized("missing credentials")
	}
	if err := s.ensureReady(); err != nil {
		return nil, err
	}
	return s.friends.ListFriendIDs(ctx, userID)
}

func (s *FriendService) Accept(ctx context.Context, userID, requestID string) (domain.FriendRequest, error) {
	if userID == "" {
		return domain.FriendRequest{}, apperr.Unauthorized("missing credentials")
	}
	if requestID == "" {
		return domain.FriendRequest{}, apperr.Invalid("request id required")
	}
	if err := s.ensureReady(); err != nil {
		return domain.FriendRequest{}, err
	}
	req, err := s.friends.AcceptRequest(ctx, requestID, userID, s.now().UTC())
	if err != nil {
		return domain.FriendRequest{}, err
	}
	return s.hydrateRequest(ctx, req)
}

func (s *FriendService) Reject(ctx context.Context, userID, requestID string) (domain.FriendRequest, error) {
	if userID == "" {
		return domain.FriendRequest{}, apperr.Unauthorized("missing credentials")
	}
	if requestID == "" {
		return domain.FriendRequest{}, apperr.Invalid("request id required")
	}
	if err := s.ensureReady(); err != nil {
		return domain.FriendRequest{}, err
	}
	req, err := s.friends.RejectRequest(ctx, requestID, userID, s.now().UTC())
	if err != nil {
		return domain.FriendRequest{}, err
	}
	return s.hydrateRequest(ctx, req)
}

func (s *FriendService) hydrateRequest(ctx context.Context, req domain.FriendRequest) (domain.FriendRequest, error) {
	from, err := s.users.FindByID(ctx, req.FromUserID)
	if err == nil {
		req.FromUser = &from
	}
	to, err := s.users.FindByID(ctx, req.ToUserID)
	if err == nil {
		req.ToUser = &to
	}
	return req, nil
}

func normalizeFriendEmail(email string) (string, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return "", apperr.Invalid("email is required")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return "", apperr.Invalid("email is invalid")
	}
	if strings.Contains(email, " ") || strings.Contains(email, "<") {
		return "", apperr.Invalid("email is invalid")
	}
	return email, nil
}
