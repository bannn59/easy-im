package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"easy-im/backend/internal/apperr"
	"easy-im/backend/internal/domain"
)

type memFriendStore struct {
	requests map[string]domain.FriendRequest
	// undirected edges keyed by canonical "a|b"
	friends map[string]struct{}
}

func newMemFriendStore() *memFriendStore {
	return &memFriendStore{
		requests: map[string]domain.FriendRequest{},
		friends:  map[string]struct{}{},
	}
}

func friendKey(a, b string) string {
	if a < b {
		return a + "|" + b
	}
	return b + "|" + a
}

func (m *memFriendStore) CreateRequest(_ context.Context, req domain.FriendRequest) error {
	for _, existing := range m.requests {
		if existing.FromUserID == req.FromUserID &&
			existing.ToUserID == req.ToUserID &&
			existing.Status == domain.FriendRequestPending {
			return apperr.Conflict("friend request already pending")
		}
	}
	m.requests[req.ID] = req
	return nil
}

func (m *memFriendStore) GetRequestByID(_ context.Context, id string) (domain.FriendRequest, error) {
	req, ok := m.requests[id]
	if !ok {
		return domain.FriendRequest{}, apperr.NotFound("friend request not found")
	}
	return req, nil
}

func (m *memFriendStore) FindPending(_ context.Context, fromUserID, toUserID string) (domain.FriendRequest, error) {
	for _, req := range m.requests {
		if req.FromUserID == fromUserID &&
			req.ToUserID == toUserID &&
			req.Status == domain.FriendRequestPending {
			return req, nil
		}
	}
	return domain.FriendRequest{}, apperr.NotFound("friend request not found")
}

func (m *memFriendStore) AreFriends(_ context.Context, userID1, userID2 string) (bool, error) {
	_, ok := m.friends[friendKey(userID1, userID2)]
	return ok, nil
}

func (m *memFriendStore) ListIncomingPending(_ context.Context, userID string) ([]domain.FriendRequest, error) {
	var out []domain.FriendRequest
	for _, req := range m.requests {
		if req.ToUserID == userID && req.Status == domain.FriendRequestPending {
			out = append(out, req)
		}
	}
	return out, nil
}

func (m *memFriendStore) ListFriends(_ context.Context, userID string) ([]domain.User, error) {
	var out []domain.User
	for k := range m.friends {
		// k is "a|b"
		var a, b string
		for i := 0; i < len(k); i++ {
			if k[i] == '|' {
				a, b = k[:i], k[i+1:]
				break
			}
		}
		peer := ""
		switch userID {
		case a:
			peer = b
		case b:
			peer = a
		default:
			continue
		}
		out = append(out, domain.User{ID: peer, Email: peer + "@x.com"})
	}
	return out, nil
}

func (m *memFriendStore) ListFriendIDs(_ context.Context, userID string) ([]string, error) {
	var out []string
	for k := range m.friends {
		var a, b string
		for i := 0; i < len(k); i++ {
			if k[i] == '|' {
				a, b = k[:i], k[i+1:]
				break
			}
		}
		switch userID {
		case a:
			out = append(out, b)
		case b:
			out = append(out, a)
		}
	}
	return out, nil
}

func (m *memFriendStore) AcceptRequest(_ context.Context, requestID, actorUserID string, respondedAt time.Time) (domain.FriendRequest, error) {
	req, ok := m.requests[requestID]
	if !ok {
		return domain.FriendRequest{}, apperr.NotFound("friend request not found")
	}
	if req.ToUserID != actorUserID {
		return domain.FriendRequest{}, apperr.Forbidden("not allowed to accept this request")
	}
	if req.Status != domain.FriendRequestPending {
		return domain.FriendRequest{}, apperr.Conflict("friend request is not pending")
	}
	req.Status = domain.FriendRequestAccepted
	req.RespondedAt = &respondedAt
	m.requests[requestID] = req
	m.friends[friendKey(req.FromUserID, req.ToUserID)] = struct{}{}
	return req, nil
}

func (m *memFriendStore) RejectRequest(_ context.Context, requestID, actorUserID string, respondedAt time.Time) (domain.FriendRequest, error) {
	req, ok := m.requests[requestID]
	if !ok {
		return domain.FriendRequest{}, apperr.NotFound("friend request not found")
	}
	if req.ToUserID != actorUserID {
		return domain.FriendRequest{}, apperr.Forbidden("not allowed to reject this request")
	}
	if req.Status != domain.FriendRequestPending {
		return domain.FriendRequest{}, apperr.Conflict("friend request is not pending")
	}
	req.Status = domain.FriendRequestRejected
	req.RespondedAt = &respondedAt
	m.requests[requestID] = req
	return req, nil
}

type memFriendUsers struct {
	byEmail map[string]domain.UserRecord
	byID    map[string]domain.User
}

func (m *memFriendUsers) FindByEmail(_ context.Context, email string) (domain.UserRecord, error) {
	rec, ok := m.byEmail[email]
	if !ok {
		return domain.UserRecord{}, apperr.NotFound("user not found")
	}
	return rec, nil
}

func (m *memFriendUsers) FindByID(_ context.Context, id string) (domain.User, error) {
	u, ok := m.byID[id]
	if !ok {
		return domain.User{}, apperr.NotFound("user not found")
	}
	return u, nil
}

func testFriendUsers() *memFriendUsers {
	now := time.Now().UTC()
	ua := domain.User{ID: "ua", Email: "a@x.com", CreatedAt: now, UpdatedAt: now}
	ub := domain.User{ID: "ub", Email: "b@x.com", CreatedAt: now, UpdatedAt: now}
	uc := domain.User{ID: "uc", Email: "c@x.com", CreatedAt: now, UpdatedAt: now}
	return &memFriendUsers{
		byEmail: map[string]domain.UserRecord{
			"a@x.com": {User: ua, PasswordHash: "x"},
			"b@x.com": {User: ub, PasswordHash: "x"},
			"c@x.com": {User: uc, PasswordHash: "x"},
		},
		byID: map[string]domain.User{
			"ua": ua,
			"ub": ub,
			"uc": uc,
		},
	}
}

func TestFriendSendListAccept(t *testing.T) {
	store := newMemFriendStore()
	users := testFriendUsers()
	svc := NewFriendService(store, users)

	req, err := svc.SendRequest(context.Background(), "ua", "b@x.com")
	if err != nil {
		t.Fatal(err)
	}
	if req.Status != domain.FriendRequestPending || req.FromUserID != "ua" || req.ToUserID != "ub" {
		t.Fatalf("unexpected request: %+v", req)
	}

	incoming, err := svc.ListIncoming(context.Background(), "ub")
	if err != nil || len(incoming) != 1 {
		t.Fatalf("incoming for b: err=%v len=%d", err, len(incoming))
	}
	if incoming[0].ID != req.ID {
		t.Fatalf("want request %s, got %s", req.ID, incoming[0].ID)
	}

	// A should not see it as incoming.
	incomingA, err := svc.ListIncoming(context.Background(), "ua")
	if err != nil || len(incomingA) != 0 {
		t.Fatalf("incoming for a: err=%v len=%d", err, len(incomingA))
	}

	accepted, err := svc.Accept(context.Background(), "ub", req.ID)
	if err != nil {
		t.Fatal(err)
	}
	if accepted.Status != domain.FriendRequestAccepted {
		t.Fatalf("want accepted, got %s", accepted.Status)
	}

	friendsA, err := svc.ListFriends(context.Background(), "ua")
	if err != nil || len(friendsA) != 1 || friendsA[0].ID != "ub" {
		t.Fatalf("friends a: err=%v list=%+v", err, friendsA)
	}
	friendsB, err := svc.ListFriends(context.Background(), "ub")
	if err != nil || len(friendsB) != 1 || friendsB[0].ID != "ua" {
		t.Fatalf("friends b: err=%v list=%+v", err, friendsB)
	}

	friendIDsA, err := svc.ListFriendIDs(context.Background(), "ua")
	if err != nil || len(friendIDsA) != 1 || friendIDsA[0] != "ub" {
		t.Fatalf("friend ids a: err=%v ids=%+v", err, friendIDsA)
	}
	friendIDsB, err := svc.ListFriendIDs(context.Background(), "ub")
	if err != nil || len(friendIDsB) != 1 || friendIDsB[0] != "ua" {
		t.Fatalf("friend ids b: err=%v ids=%+v", err, friendIDsB)
	}

	incomingAfter, err := svc.ListIncoming(context.Background(), "ub")
	if err != nil || len(incomingAfter) != 0 {
		t.Fatalf("pending should clear after accept: err=%v len=%d", err, len(incomingAfter))
	}
}

func TestFriendReject(t *testing.T) {
	store := newMemFriendStore()
	users := testFriendUsers()
	svc := NewFriendService(store, users)

	req, err := svc.SendRequest(context.Background(), "ua", "b@x.com")
	if err != nil {
		t.Fatal(err)
	}
	rejected, err := svc.Reject(context.Background(), "ub", req.ID)
	if err != nil {
		t.Fatal(err)
	}
	if rejected.Status != domain.FriendRequestRejected {
		t.Fatalf("want rejected, got %s", rejected.Status)
	}

	friendsA, err := svc.ListFriends(context.Background(), "ua")
	if err != nil || len(friendsA) != 0 {
		t.Fatalf("not friends after reject: err=%v list=%+v", err, friendsA)
	}
	friendsB, err := svc.ListFriends(context.Background(), "ub")
	if err != nil || len(friendsB) != 0 {
		t.Fatalf("not friends after reject: err=%v list=%+v", err, friendsB)
	}

	// Re-request after reject is allowed.
	req2, err := svc.SendRequest(context.Background(), "ua", "b@x.com")
	if err != nil {
		t.Fatal(err)
	}
	if req2.Status != domain.FriendRequestPending {
		t.Fatalf("want pending re-request, got %s", req2.Status)
	}
}

func TestFriendSendValidation(t *testing.T) {
	store := newMemFriendStore()
	users := testFriendUsers()
	svc := NewFriendService(store, users)
	ctx := context.Background()

	t.Run("self", func(t *testing.T) {
		_, err := svc.SendRequest(ctx, "ua", "a@x.com")
		if !errors.Is(err, apperr.ErrInvalid) {
			t.Fatalf("want invalid self, got %v", err)
		}
	})

	t.Run("unknown email", func(t *testing.T) {
		_, err := svc.SendRequest(ctx, "ua", "missing@x.com")
		if !errors.Is(err, apperr.ErrInvalid) {
			t.Fatalf("want invalid unknown email, got %v", err)
		}
	})

	t.Run("empty email", func(t *testing.T) {
		_, err := svc.SendRequest(ctx, "ua", "  ")
		if !errors.Is(err, apperr.ErrInvalid) {
			t.Fatalf("want invalid empty email, got %v", err)
		}
	})

	t.Run("duplicate pending", func(t *testing.T) {
		if _, err := svc.SendRequest(ctx, "ua", "b@x.com"); err != nil {
			t.Fatal(err)
		}
		_, err := svc.SendRequest(ctx, "ua", "b@x.com")
		if !errors.Is(err, apperr.ErrConflict) {
			t.Fatalf("want conflict duplicate pending, got %v", err)
		}
	})

	t.Run("reverse pending", func(t *testing.T) {
		store2 := newMemFriendStore()
		svc2 := NewFriendService(store2, users)
		if _, err := svc2.SendRequest(ctx, "ua", "b@x.com"); err != nil {
			t.Fatal(err)
		}
		_, err := svc2.SendRequest(ctx, "ub", "a@x.com")
		if !errors.Is(err, apperr.ErrConflict) {
			t.Fatalf("want conflict reverse pending, got %v", err)
		}
	})

	t.Run("already friends", func(t *testing.T) {
		store2 := newMemFriendStore()
		svc2 := NewFriendService(store2, users)
		req, err := svc2.SendRequest(ctx, "ua", "c@x.com")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := svc2.Accept(ctx, "uc", req.ID); err != nil {
			t.Fatal(err)
		}
		_, err = svc2.SendRequest(ctx, "ua", "c@x.com")
		if !errors.Is(err, apperr.ErrConflict) {
			t.Fatalf("want conflict already friends, got %v", err)
		}
	})
}

func TestFriendAcceptAuthz(t *testing.T) {
	store := newMemFriendStore()
	users := testFriendUsers()
	svc := NewFriendService(store, users)
	ctx := context.Background()

	req, err := svc.SendRequest(ctx, "ua", "b@x.com")
	if err != nil {
		t.Fatal(err)
	}

	// Sender cannot accept own outgoing request.
	_, err = svc.Accept(ctx, "ua", req.ID)
	if !errors.Is(err, apperr.ErrForbidden) {
		t.Fatalf("want forbidden for sender accept, got %v", err)
	}

	// Stranger cannot accept.
	_, err = svc.Accept(ctx, "uc", req.ID)
	if !errors.Is(err, apperr.ErrForbidden) {
		t.Fatalf("want forbidden for stranger accept, got %v", err)
	}

	// Sender cannot reject.
	_, err = svc.Reject(ctx, "ua", req.ID)
	if !errors.Is(err, apperr.ErrForbidden) {
		t.Fatalf("want forbidden for sender reject, got %v", err)
	}
}
