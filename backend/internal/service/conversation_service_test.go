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
	items       map[string]domain.Conversation
	members     map[string]map[string]struct{}
	lastRead    map[string]map[string]int64 // conv -> user -> seq
}

func newMemConv() *memConv {
	return &memConv{
		items:    map[string]domain.Conversation{},
		members:  map[string]map[string]struct{}{},
		lastRead: map[string]map[string]int64{},
	}
}

func (m *memConv) Create(_ context.Context, c domain.Conversation, memberIDs []string) error {
	m.items[c.ID] = c
	set := map[string]struct{}{}
	for _, id := range memberIDs {
		set[id] = struct{}{}
	}
	m.members[c.ID] = set
	m.lastRead[c.ID] = map[string]int64{}
	for _, id := range memberIDs {
		m.lastRead[c.ID][id] = 0
	}
	return nil
}

func (m *memConv) ListForUser(_ context.Context, userID string) ([]domain.Conversation, error) {
	var out []domain.Conversation
	for id, set := range m.members {
		if _, ok := set[userID]; ok {
			c := m.items[id]
			c.LastReadSeq = m.lastRead[id][userID]
			out = append(out, c)
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
	c.LastReadSeq = m.lastRead[conversationID][userID]
	return c, nil
}

func (m *memConv) MarkRead(_ context.Context, conversationID, userID string, seq int64) (int64, error) {
	set, ok := m.members[conversationID]
	if !ok {
		return 0, apperr.NotFound("conversation not found")
	}
	if _, ok := set[userID]; !ok {
		return 0, apperr.NotFound("conversation not found")
	}
	if m.lastRead[conversationID] == nil {
		m.lastRead[conversationID] = map[string]int64{}
	}
	cur := m.lastRead[conversationID][userID]
	if seq > cur {
		cur = seq
	}
	m.lastRead[conversationID][userID] = cur
	return cur, nil
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

func TestConversationMarkRead(t *testing.T) {
	users := &memEmailUsers{
		byEmail: map[string]string{"a@x.com": "ua", "b@x.com": "ub"},
		byID: map[string]domain.User{
			"ua": {ID: "ua", Email: "a@x.com"},
			"ub": {ID: "ub", Email: "b@x.com"},
		},
	}
	convs := newMemConv()
	svc := NewConversationService(convs, users)
	c, err := svc.Create(context.Background(), CreateConversationInput{
		MemberEmails:  []string{"b@x.com"},
		CreatorUserID: "ua",
	})
	if err != nil {
		t.Fatal(err)
	}
	// simulate head
	seq := int64(5)
	now := time.Now().UTC()
	item := convs.items[c.ID]
	item.LastMessageSeq = &seq
	item.LastMessageAt = &now
	preview := "hi"
	sender := "ua"
	item.LastMessagePreview = &preview
	item.LastMessageSenderID = &sender
	convs.items[c.ID] = item

	res, err := svc.MarkRead(context.Background(), c.ID, "ub", nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.LastReadSeq != 5 || res.UnreadCount != 0 {
		t.Fatalf("mark read: %+v", res)
	}
	high := int64(99)
	res, err = svc.MarkRead(context.Background(), c.ID, "ub", &high)
	if err != nil {
		t.Fatal(err)
	}
	if res.LastReadSeq != 5 {
		t.Fatalf("want clamp to head 5, got %d", res.LastReadSeq)
	}
}
