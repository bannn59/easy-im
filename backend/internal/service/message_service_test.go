package service

import (
	"context"
	"errors"
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
}

func newMemMsg() *memMsg {
	return &memMsg{
		seq:  map[string]int64{},
		byID: map[string]domain.Message{},
		cli:  map[string]string{},
		list: map[string][]domain.Message{},
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
