package app

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"easy-im/backend/internal/domain"
	"easy-im/backend/internal/hub"
	"easy-im/backend/internal/mq"
	"easy-im/backend/internal/service"
)

// fakeHub captures every event routed to users.
type fakeHub struct {
	mu     sync.Mutex
	pushed []struct {
		userIDs []string
		event   hub.Event
	}
}

func (f *fakeHub) PublishToUsers(userIDs []string, event hub.Event) {
	f.mu.Lock()
	defer f.mu.Unlock()
	// Copy slices to avoid aliasing caller buffers.
	ids := append([]string(nil), userIDs...)
	f.pushed = append(f.pushed, struct {
		userIDs []string
		event   hub.Event
	}{ids, event})
}

func (f *fakeHub) reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.pushed = nil
}

func (f *fakeHub) len() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.pushed)
}

// memMessageStore implements the slice of MessageStore used by FanoutMessage.
type memMessageStore struct {
	byID map[string]domain.Message
}

func (m *memMessageStore) Insert(_ context.Context, msg domain.Message) (domain.Message, error) { return msg, nil }
func (m *memMessageStore) List(_ context.Context, _ string, _ int64, _ int) ([]domain.Message, error) {
	return nil, nil
}
func (m *memMessageStore) FindByID(_ context.Context, id string) (domain.Message, error) {
	return m.byID[id], nil
}
func (m *memMessageStore) FindByIDs(_ context.Context, ids []string) (map[string]domain.Message, error) {
	out := map[string]domain.Message{}
	for _, id := range ids {
		if m, ok := m.byID[id]; ok {
			out[id] = m
		}
	}
	return out, nil
}
func (m *memMessageStore) UpdateBody(_ context.Context, id, body string, _ time.Time) (domain.Message, error) {
	return domain.Message{}, nil
}
func (m *memMessageStore) MarkRecalled(_ context.Context, id string, _ time.Time) (domain.Message, error) {
	return domain.Message{}, nil
}

type memMembers map[string][]string

func (m memMembers) IsMember(_ context.Context, conversationID, userID string) (bool, error) {
	for _, u := range m[conversationID] {
		if u == userID {
			return true, nil
		}
	}
	return false, nil
}

func (m memMembers) ListMemberIDs(_ context.Context, conversationID string) ([]string, error) {
	return m[conversationID], nil
}

func TestFanoutSkipsOwnOrigin(t *testing.T) {
	fh := &fakeHub{}
	opts := FanoutConsumerOpts{
		NodeID: "node1",
		Members: memMembers{"c1": {"u1", "u2"}},
		Hub:     fh,
	}

	msg := mq.Message{Topic: mq.TopicMessages, Value: mustJSON(t, mq.NewReadEvent("c1", "u2", 5, "node1"))}
	if err := FanoutHandler(context.Background(), opts, msg); err != nil {
		t.Fatal(err)
	}
	if fh.len() != 0 {
		t.Fatalf("own-origin event must be skipped, got %d pushes", fh.len())
	}
}

func TestFanoutCreatedDeliversToMembers(t *testing.T) {
	fh := &fakeHub{}
	store := &memMessageStore{byID: map[string]domain.Message{
		"m1": {
			ID: "m1", ConversationID: "c1", SenderID: "u1", Body: "hi", Seq: 3,
			CreatedAt: time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC),
		},
	}}
	msgSvc := service.NewMessageService(store, memMembers{"c1": {"u1", "u2"}}, nil)

	opts := FanoutConsumerOpts{
		NodeID: "node2",
		Members: memMembers{"c1": {"u1", "u2"}},
		Hub:     fh,
		Msg:     msgSvc,
	}
	ev := mq.NewMessageEvent(store.byID["m1"])
	ev.Origin = "node1"

	msg := mq.Message{Topic: mq.TopicMessages, Value: mustJSON(t, ev)}
	if err := FanoutHandler(context.Background(), opts, msg); err != nil {
		t.Fatal(err)
	}
	if fh.len() != 1 {
		t.Fatalf("want 1 push, got %d", fh.len())
	}
	p := fh.pushed[0]
	if len(p.userIDs) != 2 || p.event.Type != "message.created" {
		t.Fatalf("delivery mismatch: users=%v type=%q", p.userIDs, p.event.Type)
	}
	var payload map[string]any
	if err := json.Unmarshal(p.event.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["id"] != "m1" || payload["seq"] != float64(3) {
		t.Fatalf("payload mismatch: %+v", payload)
	}
}

func TestFanoutReadDelivers(t *testing.T) {
	fh := &fakeHub{}
	convSvc := service.NewConversationService(nil, nil, nil, nil)

	opts := FanoutConsumerOpts{
		NodeID:   "node2",
		Members:  memMembers{"c1": {"u1", "u2"}},
		Hub:      fh,
		Conv:     convSvc,
	}
	msg := mq.Message{Topic: mq.TopicMessages, Value: mustJSON(t, mq.NewReadEvent("c1", "u2", 7, "node1"))}
	if err := FanoutHandler(context.Background(), opts, msg); err != nil {
		t.Fatal(err)
	}
	if fh.len() != 1 {
		t.Fatalf("want 1 push, got %d", fh.len())
	}
	p := fh.pushed[0]
	if p.event.Type != "message.read" {
		t.Fatalf("want message.read, got %q", p.event.Type)
	}
	var payload map[string]any
	if err := json.Unmarshal(p.event.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	if payload["reader_id"] != "u2" || payload["last_read_seq"] != float64(7) {
		t.Fatalf("read payload mismatch: %+v", payload)
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}
