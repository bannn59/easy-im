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
	items    map[string]domain.Conversation
	members  map[string]map[string]struct{}
	lastRead map[string]map[string]int64 // conv -> user -> seq
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
			c.MemberCount = len(set)
			members := make([]domain.User, 0, len(set))
			for mid := range set {
				members = append(members, domain.User{ID: mid})
			}
			c.Members = members
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
	c.MemberCount = len(set)
	members := make([]domain.User, 0, len(set))
	for mid := range set {
		members = append(members, domain.User{ID: mid})
	}
	c.Members = members
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

func (m *memConv) FindDirectBetween(_ context.Context, userID1, userID2 string) (domain.Conversation, error) {
	var best *domain.Conversation
	for id, set := range m.members {
		if len(set) != 2 {
			continue
		}
		if _, ok := set[userID1]; !ok {
			continue
		}
		if _, ok := set[userID2]; !ok {
			continue
		}
		c := m.items[id]
		if best == nil {
			cp := c
			best = &cp
			continue
		}
		if betterDirect(c, *best) {
			cp := c
			best = &cp
		}
	}
	if best == nil {
		return domain.Conversation{}, apperr.NotFound("conversation not found")
	}
	return *best, nil
}

// betterDirect mirrors SQL: last_message_at DESC NULLS LAST, created_at DESC, id DESC.
func betterDirect(a, b domain.Conversation) bool {
	if a.LastMessageAt != nil && b.LastMessageAt == nil {
		return true
	}
	if a.LastMessageAt == nil && b.LastMessageAt != nil {
		return false
	}
	if a.LastMessageAt != nil && b.LastMessageAt != nil {
		if a.LastMessageAt.After(*b.LastMessageAt) {
			return true
		}
		if a.LastMessageAt.Before(*b.LastMessageAt) {
			return false
		}
	}
	if a.CreatedAt.After(b.CreatedAt) {
		return true
	}
	if a.CreatedAt.Before(b.CreatedAt) {
		return false
	}
	return a.ID > b.ID
}

type memConvUsers struct {
	byID map[string]domain.User
}

func (m *memConvUsers) FindByID(_ context.Context, id string) (domain.User, error) {
	u, ok := m.byID[id]
	if !ok {
		return domain.User{}, apperr.NotFound("user not found")
	}
	return u, nil
}

type memFriends struct {
	pairs map[string]struct{} // canonical "a|b" with a < b
}

func newMemFriends(pairs ...[2]string) *memFriends {
	m := &memFriends{pairs: map[string]struct{}{}}
	for _, p := range pairs {
		a, b := p[0], p[1]
		if a > b {
			a, b = b, a
		}
		m.pairs[a+"|"+b] = struct{}{}
	}
	return m
}

func (m *memFriends) AreFriends(_ context.Context, userID1, userID2 string) (bool, error) {
	a, b := userID1, userID2
	if a > b {
		a, b = b, a
	}
	_, ok := m.pairs[a+"|"+b]
	return ok, nil
}

func TestOpenDirectCreateListACL(t *testing.T) {
	users := &memConvUsers{
		byID: map[string]domain.User{
			"ua": {ID: "ua", Email: "a@x.com", CreatedAt: time.Now()},
			"ub": {ID: "ub", Email: "b@x.com", CreatedAt: time.Now()},
		},
	}
	convs := newMemConv()
	friends := newMemFriends([2]string{"ua", "ub"})
	svc := NewConversationService(convs, users, friends)

	c, err := svc.OpenDirect(context.Background(), "ua", "ub")
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
}

func TestOpenDirectIdempotent(t *testing.T) {
	users := &memConvUsers{
		byID: map[string]domain.User{
			"ua": {ID: "ua", Email: "a@x.com"},
			"ub": {ID: "ub", Email: "b@x.com"},
		},
	}
	convs := newMemConv()
	friends := newMemFriends([2]string{"ua", "ub"})
	svc := NewConversationService(convs, users, friends)

	c1, err := svc.OpenDirect(context.Background(), "ua", "ub")
	if err != nil {
		t.Fatal(err)
	}
	c2, err := svc.OpenDirect(context.Background(), "ub", "ua")
	if err != nil {
		t.Fatal(err)
	}
	if c1.ID != c2.ID {
		t.Fatalf("want same conversation id, got %s vs %s", c1.ID, c2.ID)
	}
	if len(convs.items) != 1 {
		t.Fatalf("want 1 conversation, got %d", len(convs.items))
	}
}

func TestOpenDirectRejects(t *testing.T) {
	users := &memConvUsers{
		byID: map[string]domain.User{
			"ua": {ID: "ua", Email: "a@x.com"},
			"ub": {ID: "ub", Email: "b@x.com"},
			"uc": {ID: "uc", Email: "c@x.com"},
		},
	}
	convs := newMemConv()
	friends := newMemFriends([2]string{"ua", "ub"})
	svc := NewConversationService(convs, users, friends)

	if _, err := svc.OpenDirect(context.Background(), "ua", "ua"); !errors.Is(err, apperr.ErrInvalid) {
		t.Fatalf("want invalid for self, got %v", err)
	}
	if _, err := svc.OpenDirect(context.Background(), "ua", "uc"); !errors.Is(err, apperr.ErrForbidden) {
		t.Fatalf("want forbidden for non-friend, got %v", err)
	}
	if _, err := svc.OpenDirect(context.Background(), "ua", "missing"); !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("want not found for missing peer, got %v", err)
	}
}

func TestOpenDirectPicksLatestAmongMulti(t *testing.T) {
	users := &memConvUsers{
		byID: map[string]domain.User{
			"ua": {ID: "ua", Email: "a@x.com"},
			"ub": {ID: "ub", Email: "b@x.com"},
		},
	}
	convs := newMemConv()
	friends := newMemFriends([2]string{"ua", "ub"})
	svc := NewConversationService(convs, users, friends)

	old := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	mid := time.Date(2024, 6, 1, 0, 0, 0, 0, time.UTC)
	newerMsg := time.Date(2024, 7, 1, 0, 0, 0, 0, time.UTC)

	// Older with no messages.
	_ = convs.Create(context.Background(), domain.Conversation{
		ID: "c-old", CreatedBy: "ua", CreatedAt: old, UpdatedAt: old,
	}, []string{"ua", "ub"})
	// Newer created but still no messages — would win if all have nil last_message_at.
	_ = convs.Create(context.Background(), domain.Conversation{
		ID: "c-mid", CreatedBy: "ua", CreatedAt: mid, UpdatedAt: mid,
	}, []string{"ua", "ub"})
	// Older create but has messages — should win by last_message_at.
	item := domain.Conversation{
		ID: "c-msg", CreatedBy: "ua", CreatedAt: old.Add(time.Hour), UpdatedAt: newerMsg,
		LastMessageAt: &newerMsg,
	}
	_ = convs.Create(context.Background(), item, []string{"ua", "ub"})
	// Group (3 members) must be ignored.
	_ = convs.Create(context.Background(), domain.Conversation{
		ID: "c-group", CreatedBy: "ua", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}, []string{"ua", "ub", "ux"})

	got, err := svc.OpenDirect(context.Background(), "ua", "ub")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "c-msg" {
		t.Fatalf("want c-msg (latest last_message_at), got %s", got.ID)
	}
}

func TestOpenDirectPicksLatestCreatedWhenNoMessages(t *testing.T) {
	users := &memConvUsers{
		byID: map[string]domain.User{
			"ua": {ID: "ua", Email: "a@x.com"},
			"ub": {ID: "ub", Email: "b@x.com"},
		},
	}
	convs := newMemConv()
	friends := newMemFriends([2]string{"ua", "ub"})
	svc := NewConversationService(convs, users, friends)

	t1 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2024, 2, 1, 0, 0, 0, 0, time.UTC)
	_ = convs.Create(context.Background(), domain.Conversation{
		ID: "c1", CreatedBy: "ua", CreatedAt: t1, UpdatedAt: t1,
	}, []string{"ua", "ub"})
	_ = convs.Create(context.Background(), domain.Conversation{
		ID: "c2", CreatedBy: "ua", CreatedAt: t2, UpdatedAt: t2,
	}, []string{"ua", "ub"})

	got, err := svc.OpenDirect(context.Background(), "ua", "ub")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "c2" {
		t.Fatalf("want c2 (latest created), got %s", got.ID)
	}
}

func TestConversationMarkRead(t *testing.T) {
	users := &memConvUsers{
		byID: map[string]domain.User{
			"ua": {ID: "ua", Email: "a@x.com"},
			"ub": {ID: "ub", Email: "b@x.com"},
		},
	}
	convs := newMemConv()
	friends := newMemFriends([2]string{"ua", "ub"})
	svc := NewConversationService(convs, users, friends)
	c, err := svc.OpenDirect(context.Background(), "ua", "ub")
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
