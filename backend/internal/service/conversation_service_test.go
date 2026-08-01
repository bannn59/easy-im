package service

import (
	"context"
	"errors"
	"fmt"
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

func (m *memConv) ListMemberIDs(_ context.Context, conversationID string) ([]string, error) {
	set, ok := m.members[conversationID]
	if !ok {
		return nil, nil
	}
	var out []string
	for id := range set {
		out = append(out, id)
	}
	return out, nil
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

func (m *memConv) AddMembers(_ context.Context, conversationID string, userIDs []string) error {
	set, ok := m.members[conversationID]
	if !ok {
		return apperr.NotFound("conversation not found")
	}
	for _, id := range userIDs {
		set[id] = struct{}{}
		m.lastRead[conversationID][id] = 0
	}
	return nil
}

func (m *memConv) RemoveMember(_ context.Context, conversationID, userID string) error {
	if set, ok := m.members[conversationID]; ok {
		delete(set, userID)
		delete(m.lastRead[conversationID], userID)
	}
	return nil
}

func (m *memConv) SetOwner(_ context.Context, conversationID, newOwnerID string) error {
	c, ok := m.items[conversationID]
	if !ok {
		return apperr.NotFound("conversation not found")
	}
	c.CreatedBy = newOwnerID
	m.items[conversationID] = c
	return nil
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
	svc := NewConversationService(convs, users, friends, nil)

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
	svc := NewConversationService(convs, users, friends, nil)

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
	svc := NewConversationService(convs, users, friends, nil)

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
	svc := NewConversationService(convs, users, friends, nil)

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
	svc := NewConversationService(convs, users, friends, nil)

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
	svc := NewConversationService(convs, users, friends, nil)
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

func TestConversationMarkReadPublishesEvent(t *testing.T) {
	users := &memConvUsers{
		byID: map[string]domain.User{
			"ua": {ID: "ua", Email: "a@x.com"},
			"ub": {ID: "ub", Email: "b@x.com"},
		},
	}
	convs := newMemConv()
	friends := newMemFriends([2]string{"ua", "ub"})
	pub := &fakeEventPublisher{}
	svc := NewConversationService(convs, users, friends, nil).WithReadPublisher(pub)
	c, err := svc.OpenDirect(context.Background(), "ua", "ub")
	if err != nil {
		t.Fatal(err)
	}
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

	if _, err := svc.MarkRead(context.Background(), c.ID, "ub", nil); err != nil {
		t.Fatal(err)
	}

	pub.mu.Lock()
	defer pub.mu.Unlock()
	if len(pub.reads) != 1 {
		t.Fatalf("want 1 read event, got %d", len(pub.reads))
	}
	r := pub.reads[0]
	if r[0] != c.ID || r[1] != "ub" || r[2] != "5" {
		t.Fatalf("read event mismatch: %v", r)
	}
}

func TestCreateGroup(t *testing.T) {
	conv := newMemConv()
	users := &memConvUsers{byID: map[string]domain.User{
		"u1": {ID: "u1"}, "u2": {ID: "u2"}, "u3": {ID: "u3"},
	}}
	friends := newMemFriends([2]string{"u1", "u2"}, [2]string{"u1", "u3"})
	svc := NewConversationService(conv, users, friends, nil)

	title := "Weekend Trip"
	c, err := svc.CreateGroup(context.Background(), "u1", &title, []string{"u2", "u3"})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if c.Title == nil || *c.Title != title {
		t.Fatalf("title = %v, want %q", c.Title, title)
	}
	if c.CreatedBy != "u1" {
		t.Fatalf("created_by = %q, want u1", c.CreatedBy)
	}
	if c.MemberCount != 3 {
		t.Fatalf("member_count = %d, want 3", c.MemberCount)
	}
}

func TestCreateGroupNotFriend(t *testing.T) {
	conv := newMemConv()
	users := &memConvUsers{byID: map[string]domain.User{"u1": {ID: "u1"}, "u2": {ID: "u2"}}}
	friends := newMemFriends() // u1 <-> u2 not friends
	svc := NewConversationService(conv, users, friends, nil)

	_, err := svc.CreateGroup(context.Background(), "u1", nil, []string{"u2"})
	if !errors.Is(err, apperr.ErrForbidden) {
		t.Fatalf("err = %v, want forbidden", err)
	}
}

func TestCreateGroupNoMembers(t *testing.T) {
	conv := newMemConv()
	users := &memConvUsers{byID: map[string]domain.User{"u1": {ID: "u1"}}}
	friends := newMemFriends()
	svc := NewConversationService(conv, users, friends, nil)

	_, err := svc.CreateGroup(context.Background(), "u1", nil, []string{"u1", ""})
	if !errors.Is(err, apperr.ErrInvalid) {
		t.Fatalf("err = %v, want invalid", err)
	}
}

func TestCreateGroupDedupesSelf(t *testing.T) {
	conv := newMemConv()
	users := &memConvUsers{byID: map[string]domain.User{
		"u1": {ID: "u1"}, "u2": {ID: "u2"},
	}}
	friends := newMemFriends([2]string{"u1", "u2"})
	svc := NewConversationService(conv, users, friends, nil)

	c, err := svc.CreateGroup(context.Background(), "u1", nil, []string{"u2", "u1", "u2"})
	if err != nil {
		t.Fatalf("CreateGroup: %v", err)
	}
	if c.MemberCount != 2 { // self + u2, deduped
		t.Fatalf("member_count = %d, want 2", c.MemberCount)
	}
}

func TestCreateGroupTooLarge(t *testing.T) {
	conv := newMemConv()
	users := &memConvUsers{byID: map[string]domain.User{}}
	// 50 members (self + 49 peers) is allowed; 51 is not.
	for i := 1; i <= 50; i++ {
		id := fmt.Sprintf("u%02d", i)
		users.byID[id] = domain.User{ID: id}
	}
	friends := newMemFriends()
	var pairs [][2]string
	for i := 1; i <= 50; i++ {
		pairs = append(pairs, [2]string{"self", fmt.Sprintf("u%02d", i)})
	}
	friends = newMemFriends(pairs...)
	svc := NewConversationService(conv, users, friends, nil)

	var ids []string
	for i := 1; i <= 50; i++ {
		ids = append(ids, fmt.Sprintf("u%02d", i))
	}
	_, err := svc.CreateGroup(context.Background(), "self", nil, ids)
	if !errors.Is(err, apperr.ErrInvalid) {
		t.Fatalf("err = %v, want invalid (too large)", err)
	}
}

func TestCreateGroupMemberNotFound(t *testing.T) {
	conv := newMemConv()
	users := &memConvUsers{byID: map[string]domain.User{"self": {ID: "self"}}}
	friends := newMemFriends()
	svc := NewConversationService(conv, users, friends, nil)

	_, err := svc.CreateGroup(context.Background(), "self", nil, []string{"ghost"})
	if !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("err = %v, want not found", err)
	}
}

// membersTestHarness builds a service with self/peer1/peer2 and a group.
func membersTestHarness() (*ConversationService, *memConv, string) {
	conv := newMemConv()
	users := &memConvUsers{byID: map[string]domain.User{
		"self": {ID: "self"}, "peer1": {ID: "peer1"}, "peer2": {ID: "peer2"}, "ghost": {ID: "ghost"},
	}}
	// self is friends with peer1 and peer2; peer1 is friends with peer2.
	friends := newMemFriends([2]string{"self", "peer1"}, [2]string{"self", "peer2"}, [2]string{"peer1", "peer2"})
	svc := NewConversationService(conv, users, friends, nil)

	gid := "group1"
	c := domain.Conversation{ID: gid, Title: nil, CreatedBy: "self"}
	if err := conv.Create(context.Background(), c, []string{"self", "peer1"}); err != nil {
		panic(err)
	}
	return svc, conv, gid
}

func TestAddMembers(t *testing.T) {
	svc, conv, gid := membersTestHarness()
	if err := svc.AddMembers(context.Background(), gid, "peer1", []string{"peer2"}); err != nil {
		t.Fatalf("AddMembers: %v", err)
	}
	// peer2 should now be a member.
	if _, ok := conv.members[gid]["peer2"]; !ok {
		t.Fatal("peer2 not added")
	}
}

func TestAddMembersNotFriend(t *testing.T) {
	svc, _, gid := membersTestHarness()
	// self's friend is peer1; ghost is not self's friend.
	err := svc.AddMembers(context.Background(), gid, "self", []string{"ghost"})
	if !errors.Is(err, apperr.ErrForbidden) {
		t.Fatalf("err = %v, want forbidden", err)
	}
}

func TestAddMembersNonMemberRejected(t *testing.T) {
	svc, _, gid := membersTestHarness()
	// peer2 is not in the group, cannot add.
	err := svc.AddMembers(context.Background(), gid, "peer2", []string{"ghost"})
	if !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("err = %v, want not found (non-member)", err)
	}
}

func TestLeaveGroup(t *testing.T) {
	svc, conv, gid := membersTestHarness()
	if err := svc.LeaveGroup(context.Background(), gid, "peer1"); err != nil {
		t.Fatalf("LeaveGroup: %v", err)
	}
	if _, ok := conv.members[gid]["peer1"]; ok {
		t.Fatal("peer1 still member")
	}
	if _, ok := conv.members[gid]["self"]; !ok {
		t.Fatal("self should remain")
	}
}

func TestLeaveGroupOwnerForbidden(t *testing.T) {
	svc, _, gid := membersTestHarness()
	err := svc.LeaveGroup(context.Background(), gid, "self")
	if !errors.Is(err, apperr.ErrConflict) {
		t.Fatalf("err = %v, want conflict (owner must transfer)", err)
	}
}

func TestKickMember(t *testing.T) {
	svc, conv, gid := membersTestHarness()
	if err := svc.KickMember(context.Background(), gid, "self", "peer1"); err != nil {
		t.Fatalf("KickMember: %v", err)
	}
	if _, ok := conv.members[gid]["peer1"]; ok {
		t.Fatal("peer1 not kicked")
	}
}

func TestKickMemberNonOwnerForbidden(t *testing.T) {
	svc, _, gid := membersTestHarness()
	err := svc.KickMember(context.Background(), gid, "peer1", "self")
	if !errors.Is(err, apperr.ErrForbidden) {
		t.Fatalf("err = %v, want forbidden (non-owner)", err)
	}
}

func TestKickMemberSelfRejected(t *testing.T) {
	svc, _, gid := membersTestHarness()
	err := svc.KickMember(context.Background(), gid, "self", "self")
	if !errors.Is(err, apperr.ErrInvalid) {
		t.Fatalf("err = %v, want invalid (kick self)", err)
	}
}

func TestTransferOwner(t *testing.T) {
	svc, conv, gid := membersTestHarness()
	if err := svc.TransferOwner(context.Background(), gid, "self", "peer1"); err != nil {
		t.Fatalf("TransferOwner: %v", err)
	}
	if conv.items[gid].CreatedBy != "peer1" {
		t.Fatalf("owner = %q, want peer1", conv.items[gid].CreatedBy)
	}
	// Old owner can now leave.
	if err := svc.LeaveGroup(context.Background(), gid, "self"); err != nil {
		t.Fatalf("old owner leave after transfer: %v", err)
	}
}

func TestTransferOwnerNonMember(t *testing.T) {
	svc, _, gid := membersTestHarness()
	err := svc.TransferOwner(context.Background(), gid, "self", "ghost")
	if !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("err = %v, want not found (ghost not member)", err)
	}
}

func TestAddMembersAlreadyInGroup(t *testing.T) {
	svc, _, gid := membersTestHarness()
	// peer1 is already a member; adding it must conflict.
	err := svc.AddMembers(context.Background(), gid, "self", []string{"peer1"})
	if !errors.Is(err, apperr.ErrConflict) {
		t.Fatalf("err = %v, want conflict (already in group)", err)
	}
}
