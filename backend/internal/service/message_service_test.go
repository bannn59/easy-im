package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"

	"easy-im/backend/internal/apperr"
	"easy-im/backend/internal/domain"
)

type memMsg struct {
	mu   sync.Mutex
	seq  map[string]int64
	byID map[string]domain.Message
	cli  map[string]string // sender|client -> id
	list map[string][]domain.Message
	// members maps conversation -> member user set, used by GlobalSearch ACL.
	members map[string]map[string]bool
}

func newMemMsg() *memMsg {
	return &memMsg{
		seq:     map[string]int64{},
		byID:    map[string]domain.Message{},
		cli:     map[string]string{},
		list:    map[string][]domain.Message{},
		members: map[string]map[string]bool{},
	}
}

func (m *memMsg) Insert(_ context.Context, msg domain.Message) (domain.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	key := msg.SenderID + "|" + msg.ClientMsgID
	if id, ok := m.cli[key]; ok {
		return m.byID[id], nil
	}
	m.seq[msg.ConversationID]++
	msg.Seq = m.seq[msg.ConversationID]
	m.byID[msg.ID] = msg
	m.cli[key] = msg.ID
	m.list[msg.ConversationID] = append(m.list[msg.ConversationID], msg)
	return msg, nil
}

func (m *memMsg) List(_ context.Context, conversationID string, beforeSeq int64, limit int) ([]domain.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	all := m.list[conversationID]
	var filtered []domain.Message
	for _, msg := range all {
		if beforeSeq <= 0 || msg.Seq < beforeSeq {
			filtered = append(filtered, msg)
		}
	}
	if limit <= 0 {
		limit = 50
	}
	if len(filtered) > limit {
		filtered = filtered[len(filtered)-limit:]
	}
	return filtered, nil
}

func (m *memMsg) Search(_ context.Context, conversationID, query string, beforeSeq int64, limit int) ([]domain.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.Message
	for _, msg := range m.list[conversationID] {
		if msg.RecalledAt != nil {
			continue
		}
		if !strings.Contains(strings.ToLower(msg.Body), strings.ToLower(query)) {
			continue
		}
		if beforeSeq > 0 && msg.Seq >= beforeSeq {
			continue
		}
		out = append(out, msg)
	}
	// newest first
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (m *memMsg) ListAround(_ context.Context, conversationID string, aroundSeq int64, window int) ([]domain.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.Message
	for _, msg := range m.list[conversationID] {
		if msg.Seq >= aroundSeq-int64(window) && msg.Seq <= aroundSeq+int64(window) {
			out = append(out, msg)
		}
	}
	// seq ascending
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].Seq < out[i].Seq {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out, nil
}

func (m *memMsg) GlobalSearch(_ context.Context, userID, query string, cursor *domain.SearchCursor, limit int) ([]domain.GlobalSearchResult, *domain.SearchCursor, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// conversations map (mock: conv -> member set)
	var out []domain.GlobalSearchResult
	for convID, msgs := range m.list {
		members := m.members[convID]
		if _, ok := members[userID]; !ok {
			continue
		}
		for _, msg := range msgs {
			if msg.RecalledAt != nil {
				continue
			}
			if !strings.Contains(strings.ToLower(msg.Body), strings.ToLower(query)) {
				continue
			}
			out = append(out, domain.GlobalSearchResult{Message: msg})
		}
	}
	// sort by created_at desc, id desc
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].Message.CreatedAt.After(out[i].Message.CreatedAt) {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil, nil
}

func (m *memMsg) FindByID(_ context.Context, id string) (domain.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	msg, ok := m.byID[id]
	if !ok {
		return domain.Message{}, apperr.NotFound("message not found")
	}
	return msg, nil
}

func (m *memMsg) FindByIDs(_ context.Context, ids []string) (map[string]domain.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[string]domain.Message, len(ids))
	for _, id := range ids {
		if msg, ok := m.byID[id]; ok {
			out[id] = msg
		}
	}
	return out, nil
}

func (m *memMsg) UpdateBody(_ context.Context, id, body string, editedAt time.Time) (domain.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	msg, ok := m.byID[id]
	if !ok {
		return domain.Message{}, apperr.NotFound("message not found")
	}
	msg.Body = body
	msg.EditedAt = &editedAt
	m.byID[id] = msg
	return msg, nil
}

func (m *memMsg) MarkRecalled(_ context.Context, id string, recalledAt time.Time) (domain.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	msg, ok := m.byID[id]
	if !ok {
		return domain.Message{}, apperr.NotFound("message not found")
	}
	msg.RecalledAt = &recalledAt
	m.byID[id] = msg
	// Refresh the list slice so Search/ListAround observe the recall.
	for conv := range m.list {
		for i := range m.list[conv] {
			if m.list[conv][i].ID == id {
				m.list[conv][i] = msg
			}
		}
	}
	return msg, nil
}

type memMembers map[string]map[string]bool // conv -> user -> ok

func (m memMembers) IsMember(_ context.Context, conversationID, userID string) (bool, error) {
	return m[conversationID][userID], nil
}

func (m memMembers) ListMemberIDs(_ context.Context, conversationID string) ([]string, error) {
	var out []string
	for uid, ok := range m[conversationID] {
		if ok {
			out = append(out, uid)
		}
	}
	return out, nil
}

func TestMessageSendListIdempotent(t *testing.T) {
	store := newMemMsg()
	members := memMembers{"c1": {"u1": true, "u2": true}}
	svc := NewMessageService(store, members, nil)
	svc.now = func() time.Time { return time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC) }

	m1, err := svc.Send(context.Background(), SendMessageInput{
		ConversationID: "c1", SenderID: "u1", Body: "hello", ClientMsgID: "cli-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	m2, err := svc.Send(context.Background(), SendMessageInput{
		ConversationID: "c1", SenderID: "u1", Body: "hello", ClientMsgID: "cli-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if m1.Message.ID != m2.Message.ID || m1.Message.Seq != m2.Message.Seq {
		t.Fatalf("idempotent mismatch %+v vs %+v", m1, m2)
	}
	list, err := svc.List(context.Background(), "c1", "u2", 0, 50)
	if err != nil || len(list) != 1 {
		t.Fatalf("list: %v len=%d", err, len(list))
	}
	if _, err := svc.Send(context.Background(), SendMessageInput{
		ConversationID: "c1", SenderID: "stranger", Body: "x", ClientMsgID: "c",
	}); !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("want not found, got %v", err)
	}
}

func TestMessageReply(t *testing.T) {
	store := newMemMsg()
	members := memMembers{
		"c1": {"u1": true, "u2": true},
		"c2": {"u1": true},
	}
	svc := NewMessageService(store, members, nil)
	svc.now = func() time.Time { return time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC) }

	base, err := svc.Send(context.Background(), SendMessageInput{
		ConversationID: "c1", SenderID: "u1", Body: "root", ClientMsgID: "r1",
	})
	if err != nil {
		t.Fatal(err)
	}

	// Valid reply
	reply, err := svc.Send(context.Background(), SendMessageInput{
		ConversationID:   "c1",
		SenderID:         "u2",
		Body:             "re",
		ClientMsgID:      "r2",
		ReplyToMessageID: base.Message.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if reply.ReplyTo == nil || reply.ReplyTo.ID != base.Message.ID || reply.ReplyTo.Body != "root" {
		t.Fatalf("reply preview: %+v", reply.ReplyTo)
	}
	if reply.Message.ReplyToMessageID == nil || *reply.Message.ReplyToMessageID != base.Message.ID {
		t.Fatalf("stored reply id: %v", reply.Message.ReplyToMessageID)
	}

	list, err := svc.List(context.Background(), "c1", "u1", 0, 50)
	if err != nil || len(list) != 2 {
		t.Fatalf("list: %v len=%d", err, len(list))
	}
	if list[1].ReplyTo == nil || list[1].ReplyTo.ID != base.Message.ID {
		t.Fatalf("list reply: %+v", list[1].ReplyTo)
	}

	// Missing target
	if _, err := svc.Send(context.Background(), SendMessageInput{
		ConversationID: "c1", SenderID: "u1", Body: "x", ClientMsgID: "bad1", ReplyToMessageID: "no-such",
	}); !errors.Is(err, apperr.ErrInvalid) {
		t.Fatalf("want invalid missing, got %v", err)
	}

	// Cross-conversation
	other, err := svc.Send(context.Background(), SendMessageInput{
		ConversationID: "c2", SenderID: "u1", Body: "other", ClientMsgID: "o1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Send(context.Background(), SendMessageInput{
		ConversationID: "c1", SenderID: "u1", Body: "x", ClientMsgID: "bad2", ReplyToMessageID: other.Message.ID,
	}); !errors.Is(err, apperr.ErrInvalid) {
		t.Fatalf("want invalid cross-conv, got %v", err)
	}
}

func TestReplyPreviewTruncation(t *testing.T) {
	long := ""
	for i := 0; i < 200; i++ {
		long += "字"
	}
	store := newMemMsg()
	members := memMembers{"c1": {"u1": true}}
	svc := NewMessageService(store, members, nil)
	base, err := svc.Send(context.Background(), SendMessageInput{
		ConversationID: "c1", SenderID: "u1", Body: long, ClientMsgID: "long1",
	})
	if err != nil {
		t.Fatal(err)
	}
	reply, err := svc.Send(context.Background(), SendMessageInput{
		ConversationID: "c1", SenderID: "u1", Body: "ok", ClientMsgID: "long2", ReplyToMessageID: base.Message.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if reply.ReplyTo == nil || utf8.RuneCountInString(reply.ReplyTo.Body) != replyPreviewMaxRunes {
		t.Fatalf("preview runes=%d body=%q", utf8.RuneCountInString(reply.ReplyTo.Body), reply.ReplyTo.Body)
	}
}

func TestMessageEdit(t *testing.T) {
	store := newMemMsg()
	members := memMembers{"c1": {"u1": true, "u2": true}}
	svc := NewMessageService(store, members, nil)
	svc.now = func() time.Time { return time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC) }
	ctx := context.Background()

	m, err := svc.Send(ctx, SendMessageInput{ConversationID: "c1", SenderID: "u1", Body: "hello", ClientMsgID: "cli-1"})
	if err != nil {
		t.Fatal(err)
	}
	id := m.Message.ID

	// Successful edit.
	edited, err := svc.Edit(ctx, "c1", id, "u1", "  edited  ")
	if err != nil {
		t.Fatal(err)
	}
	if edited.Message.Body != "edited" {
		t.Fatalf("body = %q want edited", edited.Message.Body)
	}
	if edited.Message.EditedAt == nil {
		t.Fatal("want edited_at set")
	}

	// Empty body rejected.
	if _, err := svc.Edit(ctx, "c1", id, "u1", "  "); !errors.Is(err, apperr.ErrInvalid) {
		t.Fatalf("want invalid for empty body, got %v", err)
	}

	// Not own message rejected.
	if _, err := svc.Edit(ctx, "c1", id, "u2", "x"); !errors.Is(err, apperr.ErrForbidden) {
		t.Fatalf("want forbidden for other's message, got %v", err)
	}

	// Non-member rejected.
	if _, err := svc.Edit(ctx, "c1", id, "stranger", "x"); !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("want not found for stranger, got %v", err)
	}
}

func TestMessageEditWindowExpired(t *testing.T) {
	store := newMemMsg()
	members := memMembers{"c1": {"u1": true, "u2": true}}
	svc := NewMessageService(store, members, nil)
	svc.now = func() time.Time { return time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC) }
	ctx := context.Background()

	m, err := svc.Send(ctx, SendMessageInput{ConversationID: "c1", SenderID: "u1", Body: "hello", ClientMsgID: "cli-1"})
	if err != nil {
		t.Fatal(err)
	}
	id := m.Message.ID

	// Advance beyond the 5-minute window.
	svc.now = func() time.Time { return time.Date(2026, 7, 28, 12, 10, 0, 0, time.UTC) }
	if _, err := svc.Edit(ctx, "c1", id, "u1", "too late"); !errors.Is(err, apperr.ErrInvalid) {
		t.Fatalf("want invalid after window, got %v", err)
	}
	if _, err := svc.Recall(ctx, "c1", id, "u1"); !errors.Is(err, apperr.ErrInvalid) {
		t.Fatalf("want invalid after window for recall, got %v", err)
	}
}

func TestMessageRecall(t *testing.T) {
	store := newMemMsg()
	members := memMembers{"c1": {"u1": true, "u2": true}}
	svc := NewMessageService(store, members, nil)
	svc.now = func() time.Time { return time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC) }
	ctx := context.Background()

	m, err := svc.Send(ctx, SendMessageInput{ConversationID: "c1", SenderID: "u1", Body: "secret", ClientMsgID: "cli-1"})
	if err != nil {
		t.Fatal(err)
	}
	id := m.Message.ID

	recalled, err := svc.Recall(ctx, "c1", id, "u1")
	if err != nil {
		t.Fatal(err)
	}
	if recalled.Message.RecalledAt == nil {
		t.Fatal("want recalled_at set")
	}

	// Double recall rejected.
	if _, err := svc.Recall(ctx, "c1", id, "u1"); !errors.Is(err, apperr.ErrInvalid) {
		t.Fatalf("want invalid for double recall, got %v", err)
	}

	// Editing a recalled message rejected.
	if _, err := svc.Edit(ctx, "c1", id, "u1", "new"); !errors.Is(err, apperr.ErrInvalid) {
		t.Fatalf("want invalid for edit after recall, got %v", err)
	}

	// Other member cannot recall.
	m2, err := svc.Send(ctx, SendMessageInput{ConversationID: "c1", SenderID: "u2", Body: "mine", ClientMsgID: "cli-2"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Recall(ctx, "c1", m2.Message.ID, "u1"); !errors.Is(err, apperr.ErrForbidden) {
		t.Fatalf("want forbidden for recalling other's message, got %v", err)
	}
}

// fakeEventPublisher records the bus events the service publishes.
type fakeEventPublisher struct {
	mu      sync.Mutex
	created []domain.Message
	edited  []domain.Message
	recalled []domain.Message
	reads   [][3]string // conversationID, userID, seq
}

func (f *fakeEventPublisher) PublishMessageCreated(_ context.Context, m domain.Message) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.created = append(f.created, m)
	return nil
}
func (f *fakeEventPublisher) PublishMessageEdited(_ context.Context, m domain.Message) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.edited = append(f.edited, m)
	return nil
}
func (f *fakeEventPublisher) PublishMessageRecalled(_ context.Context, m domain.Message) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.recalled = append(f.recalled, m)
	return nil
}
func (f *fakeEventPublisher) PublishMessageRead(_ context.Context, conversationID, userID string, seq int64) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads = append(f.reads, [3]string{conversationID, userID, strconv.FormatInt(seq, 10)})
	return nil
}

func TestMessageSendEditRecallPublishEvents(t *testing.T) {
	store := newMemMsg()
	members := memMembers{"c1": {"u1": true, "u2": true}}
	pub := &fakeEventPublisher{}
	svc := NewMessageService(store, members, nil).WithEventPublisher(pub)
	svc.now = func() time.Time { return time.Date(2026, 7, 28, 12, 0, 0, 0, time.UTC) }
	ctx := context.Background()

	m, err := svc.Send(ctx, SendMessageInput{ConversationID: "c1", SenderID: "u1", Body: "hello", ClientMsgID: "cli-1"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Edit(ctx, "c1", m.Message.ID, "u1", "updated"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Recall(ctx, "c1", m.Message.ID, "u1"); err != nil {
		t.Fatal(err)
	}

	pub.mu.Lock()
	defer pub.mu.Unlock()
	if len(pub.created) != 1 || len(pub.edited) != 1 || len(pub.recalled) != 1 {
		t.Fatalf("events: created=%d edited=%d recalled=%d", len(pub.created), len(pub.edited), len(pub.recalled))
	}
	if pub.created[0].ID != m.Message.ID || pub.edited[0].ID != m.Message.ID || pub.recalled[0].ID != m.Message.ID {
		t.Fatal("event message id mismatch")
	}
	if pub.edited[0].EditedAt == nil {
		t.Fatal("edited event missing edited_at")
	}
	if pub.recalled[0].RecalledAt == nil {
		t.Fatal("recalled event missing recalled_at")
	}
}

func TestMessageSearch(t *testing.T) {
	store := newMemMsg()
	members := memMembers{"c1": {"u1": true, "u2": true}}
	svc := NewMessageService(store, members, nil)
	ctx := context.Background()

	// Seed messages via Send (mock seq starts at 1: seqs 1..4).
	send := func(body, sender string) domain.Message {
		m, err := svc.Send(ctx, SendMessageInput{ConversationID: "c1", SenderID: sender, Body: body, ClientMsgID: body})
		if err != nil {
			t.Fatal(err)
		}
		return m.Message
	}
	send("hello world", "u1")       // seq 1
	send("goodbye planet", "u2")    // seq 2
	send("Hello again", "u1")       // seq 3
	recalled := send("secret hello", "u2") // seq 4
	if _, err := svc.Recall(ctx, "c1", recalled.ID, "u2"); err != nil {
		t.Fatal(err)
	}

	// Case-insensitive match, recalled excluded.
	res, err := svc.Search(ctx, "c1", "u1", "hello", 0, 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 2 {
		t.Fatalf("want 2 hits, got %d", len(res))
	}
	// Newest first: seq 3 then seq 1 (seq 4 recalled excluded).
	if res[0].Message.Seq != 3 || res[1].Message.Seq != 1 {
		t.Fatalf("seq order wrong: %d %d", res[0].Message.Seq, res[1].Message.Seq)
	}
}

func TestMessageSearchBlankQuery(t *testing.T) {
	store := newMemMsg()
	members := memMembers{"c1": {"u1": true}}
	svc := NewMessageService(store, members, nil)
	_, err := svc.Search(context.Background(), "c1", "u1", "   ", 0, 50)
	if !errors.Is(err, apperr.ErrInvalid) {
		t.Fatalf("err = %v, want invalid", err)
	}
}

func TestMessageSearchNonMember(t *testing.T) {
	store := newMemMsg()
	members := memMembers{"c1": {"u1": true}}
	svc := NewMessageService(store, members, nil)
	_, err := svc.Search(context.Background(), "c1", "ghost", "hello", 0, 50)
	if !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("err = %v, want not found", err)
	}
}

func TestMessageListAround(t *testing.T) {
	store := newMemMsg()
	members := memMembers{"c1": {"u1": true}}
	svc := NewMessageService(store, members, nil)
	ctx := context.Background()

	var ids []string
	for i := 0; i < 10; i++ {
		m, err := svc.Send(ctx, SendMessageInput{ConversationID: "c1", SenderID: "u1", Body: "msg " + strconv.Itoa(i), ClientMsgID: "cli-" + strconv.Itoa(i)})
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, m.Message.ID)
	}
	// Recall one inside the window (seqs 1..10; ids[4] is seq 5) to verify
	// ListAround includes recalled messages for jump positioning.
	if _, err := svc.Recall(ctx, "c1", ids[4], "u1"); err != nil {
		t.Fatal(err)
	}

	res, err := svc.ListAround(ctx, "c1", "u1", 5, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 5 {
		t.Fatalf("want window size 5 (seq 3..7), got %d", len(res))
	}
	// Ascending seq, includes the recalled seq 5.
	if res[0].Message.Seq != 3 || res[len(res)-1].Message.Seq != 7 {
		t.Fatalf("window bounds wrong: first=%d last=%d", res[0].Message.Seq, res[len(res)-1].Message.Seq)
	}
	foundRecalled := false
	for _, v := range res {
		if v.Message.Seq == 5 && v.Message.RecalledAt != nil {
			foundRecalled = true
		}
	}
	if !foundRecalled {
		t.Fatal("ListAround should include recalled message")
	}
}

func TestMessageListAroundNonMember(t *testing.T) {
	store := newMemMsg()
	members := memMembers{"c1": {"u1": true}}
	svc := NewMessageService(store, members, nil)
	_, err := svc.ListAround(context.Background(), "c1", "ghost", 5, 2)
	if !errors.Is(err, apperr.ErrNotFound) {
		t.Fatalf("err = %v, want not found", err)
	}
}

func TestGlobalSearchAcrossConversations(t *testing.T) {
	store := newMemMsg()
	members := memMembers{
		"c1": {"u1": true, "u2": true},
		"c2": {"u1": true},
		"c3": {"u2": true}, // u1 NOT a member -> results must be excluded
	}
	svc := NewMessageService(store, members, nil)
	ctx := context.Background()

	// Seed members on the store for GlobalSearch ACL.
	store.members = map[string]map[string]bool{
		"c1": {"u1": true, "u2": true},
		"c2": {"u1": true},
		"c3": {"u2": true},
	}

	send := func(conv, body, sender string) {
		if _, err := svc.Send(ctx, SendMessageInput{ConversationID: conv, SenderID: sender, Body: body, ClientMsgID: conv + "-" + body}); err != nil {
			t.Fatal(err)
		}
	}
	send("c1", "hello from c1", "u1")
	send("c2", "hello from c2", "u1")
	send("c3", "hello from c3 u1 not member", "u2") // u1 not in c3

	// Recall one hit in c1 to verify exclusion.
	res, _, err := svc.GlobalSearch(ctx, "u1", "hello", nil, 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 2 {
		t.Fatalf("want 2 hits (c1+c2), got %d", len(res))
	}
	convs := map[string]bool{}
	for _, r := range res {
		convs[r.Message.Message.ConversationID] = true
	}
	if !convs["c1"] || !convs["c2"] {
		t.Fatalf("missing expected conversations: %v", convs)
	}
	if convs["c3"] {
		t.Fatal("c3 result leaked despite u1 not being a member")
	}
}

func TestGlobalSearchBlankQuery(t *testing.T) {
	store := newMemMsg()
	members := memMembers{"c1": {"u1": true}}
	svc := NewMessageService(store, members, nil)
	_, _, err := svc.GlobalSearch(context.Background(), "u1", "  ", nil, 50)
	if !errors.Is(err, apperr.ErrInvalid) {
		t.Fatalf("err = %v, want invalid", err)
	}
}

func TestGlobalSearchExcludesRecalled(t *testing.T) {
	store := newMemMsg()
	members := memMembers{"c1": {"u1": true}}
	svc := NewMessageService(store, members, nil)
	store.members = map[string]map[string]bool{"c1": {"u1": true}}
	ctx := context.Background()

	m, err := svc.Send(ctx, SendMessageInput{ConversationID: "c1", SenderID: "u1", Body: "secret hello", ClientMsgID: "secret"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Recall(ctx, "c1", m.Message.ID, "u1"); err != nil {
		t.Fatal(err)
	}
	res, _, err := svc.GlobalSearch(ctx, "u1", "hello", nil, 50)
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 0 {
		t.Fatalf("recalled message should be excluded, got %d hits", len(res))
	}
}
