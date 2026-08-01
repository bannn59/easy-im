package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"easy-im/backend/internal/apperr"
	"easy-im/backend/internal/domain"
	"easy-im/backend/internal/service"
)

// memConvForHandler is a minimal ConversationStore for handler tests.
type memConvForHandler struct {
	items   map[string]domain.Conversation
	members map[string]map[string]struct{}
}

func (m *memConvForHandler) Create(_ context.Context, c domain.Conversation, memberIDs []string) error {
	m.items[c.ID] = c
	set := map[string]struct{}{}
	for _, id := range memberIDs {
		set[id] = struct{}{}
	}
	m.members[c.ID] = set
	return nil
}

func (m *memConvForHandler) ListForUser(_ context.Context, userID string) ([]domain.Conversation, error) {
	return nil, nil
}

func (m *memConvForHandler) GetIfMember(_ context.Context, conversationID, userID string) (domain.Conversation, error) {
	set, ok := m.members[conversationID]
	if !ok {
		return domain.Conversation{}, apperr.NotFound("conversation not found")
	}
	if _, ok := set[userID]; !ok {
		return domain.Conversation{}, apperr.NotFound("conversation not found")
	}
	c := m.items[conversationID]
	c.MemberCount = len(set)
	return c, nil
}

func (m *memConvForHandler) MarkRead(_ context.Context, conversationID, userID string, seq int64) (int64, error) {
	return 0, nil
}

func (m *memConvForHandler) FindDirectBetween(_ context.Context, userID1, userID2 string) (domain.Conversation, error) {
	return domain.Conversation{}, apperr.NotFound("not found")
}

func (m *memConvForHandler) ListMemberIDs(_ context.Context, conversationID string) ([]string, error) {
	return nil, nil
}

// memUsersForHandler resolves users by id.
type memUsersForHandler struct {
	byID map[string]domain.User
}

func (m *memUsersForHandler) FindByID(_ context.Context, id string) (domain.User, error) {
	u, ok := m.byID[id]
	if !ok {
		return domain.User{}, apperr.NotFound("user not found")
	}
	return u, nil
}

// memFriendsForHandler reports friendship.
type memFriendsForHandler struct {
	pairs map[string]struct{}
}

func (m *memFriendsForHandler) AreFriends(_ context.Context, userID1, userID2 string) (bool, error) {
	a, b := userID1, userID2
	if a > b {
		a, b = b, a
	}
	_, ok := m.pairs[a+"|"+b]
	return ok, nil
}

func groupHarness() *ConversationHandler {
	conv := &memConvForHandler{
		items:   map[string]domain.Conversation{},
		members: map[string]map[string]struct{}{},
	}
	users := &memUsersForHandler{byID: map[string]domain.User{
		"self": {ID: "self"}, "peer1": {ID: "peer1"}, "stranger": {ID: "stranger"},
	}}
	friends := &memFriendsForHandler{pairs: map[string]struct{}{"peer1|self": {}}}
	svc := service.NewConversationService(conv, users, friends, nil)
	return &ConversationHandler{Conv: svc}
}

func groupReq(method, path, body, userID string) *http.Request {
	var rdr io.Reader
	if body != "" {
		rdr = bytes.NewBufferString(body)
	}
	req := httptest.NewRequest(method, path, rdr)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if userID != "" {
		req = req.WithContext(context.WithValue(req.Context(), authUserKey{}, userID))
	}
	return req
}

func TestCreateGroupHandlerSuccess(t *testing.T) {
	h := groupHarness()
	req := groupReq(http.MethodPost, "/v1/conversations/groups", `{"title":"Trip","member_ids":["peer1"]}`, "self")
	rr := httptest.NewRecorder()
	h.CreateGroup(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body %s", rr.Code, rr.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if out["title"] != "Trip" {
		t.Fatalf("title = %v", out["title"])
	}
	if out["created_by"] != "self" {
		t.Fatalf("created_by = %v", out["created_by"])
	}
}

func TestCreateGroupHandlerMissingMembers(t *testing.T) {
	h := groupHarness()
	req := groupReq(http.MethodPost, "/v1/conversations/groups", `{"title":"Trip"}`, "self")
	rr := httptest.NewRecorder()
	h.CreateGroup(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}

func TestCreateGroupHandlerNotFriend(t *testing.T) {
	h := groupHarness()
	req := groupReq(http.MethodPost, "/v1/conversations/groups", `{"member_ids":["stranger"]}`, "self")
	rr := httptest.NewRecorder()
	h.CreateGroup(rr, req)

	if rr.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", rr.Code)
	}
}
