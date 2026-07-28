package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"easy-im/backend/internal/apperr"
	"easy-im/backend/internal/domain"
)

type memConv struct {
	items   map[string]domain.Conversation
	members map[string]map[string]struct{}
}

func newMemConv() *memConv {
	return &memConv{
		items:   map[string]domain.Conversation{},
		members: map[string]map[string]struct{}{},
	}
}

func (m *memConv) Create(_ context.Context, c domain.Conversation, memberIDs []string) error {
	m.items[c.ID] = c
	set := map[string]struct{}{}
	for _, id := range memberIDs {
		set[id] = struct{}{}
	}
	m.members[c.ID] = set
	return nil
}

func (m *memConv) ListForUser(_ context.Context, userID string) ([]domain.Conversation, error) {
	var out []domain.Conversation
	for id, set := range m.members {
		if _, ok := set[userID]; ok {
			out = append(out, m.items[id])
		}
	}
	return out, nil
}

func (m *memConv) GetIfMember(_ context.Context, conversationID, userID string) (domain.Conversation, error) {
	set, ok := m.members[conversationID]
	if !ok {
		return domain.Conversation{}, apperr.NotFound("conversation not found")
	}
	if _, ok := set[userID]; !ok {
		return domain.Conversation{}, apperr.NotFound("conversation not found")
	}
	c := m.items[conversationID]
	return c, nil
}

type memEmailUsers struct {
	byEmail map[string]string
	byID    map[string]domain.User
}

func (m *memEmailUsers) FindIDsByEmails(_ context.Context, emails []string) (map[string]string, error) {
	out := map[string]string{}
	for _, e := range emails {
		if id, ok := m.byEmail[e]; ok {
			out[e] = id
		}
	}
	return out, nil
}

func (m *memEmailUsers) FindByID(_ context.Context, id string) (domain.User, error) {
	u, ok := m.byID[id]
	if !ok {
		return domain.User{}, apperr.NotFound("user not found")
	}
	return u, nil
}

func TestConversationCreateListACL(t *testing.T) {
	users := &memEmailUsers{
		byEmail: map[string]string{"a@x.com": "ua", "b@x.com": "ub"},
		byID: map[string]domain.User{
			"ua": {ID: "ua", Email: "a@x.com", CreatedAt: time.Now()},
			"ub": {ID: "ub", Email: "b@x.com", CreatedAt: time.Now()},
		},
	}
	convs := newMemConv()
	svc := NewConversationService(convs, users)

	c, err := svc.Create(context.Background(), CreateConversationInput{
		Title:         "hi",
		MemberEmails:  []string{"b@x.com"},
		CreatorUserID: "ua",
	})
	if err != nil {
		t.Fatal(err)
	}
	list, err := svc.List(context.Background(), "ub")
	if err != nil || len(list) != 1 {
		t.Fatalf("list for b: %v len=%d", err, len(list))
	}
	if _, err := svc.Get(context.Background(), c.ID, "ua"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Get(context.Background(), c.ID, "stranger"); !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("want not found for stranger, got %v", err)
	}
	if _, err := svc.Create(context.Background(), CreateConversationInput{
		MemberEmails:  []string{"missing@x.com"},
		CreatorUserID: "ua",
	}); !errors.Is(err, apperr.ErrInvalid) {
		t.Fatalf("want invalid for missing email, got %v", err)
	}
}
