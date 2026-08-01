package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	"easy-im/backend/internal/apperr"
	"easy-im/backend/internal/domain"
	"easy-im/backend/internal/service"
)

// memMsgStore implements MessageStore for handler tests.
type memMsgStore struct {
	mu   sync.Mutex
	byID map[string]domain.Message
	list map[string][]domain.Message // conv -> messages, ascending seq
	seq  map[string]int64
}

func newMemMsgStore() *memMsgStore {
	return &memMsgStore{
		byID: map[string]domain.Message{},
		list: map[string][]domain.Message{},
		seq:  map[string]int64{},
	}
}

func (m *memMsgStore) Insert(_ context.Context, msg domain.Message) (domain.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.seq[msg.ConversationID]++
	msg.Seq = m.seq[msg.ConversationID]
	m.byID[msg.ID] = msg
	m.list[msg.ConversationID] = append(m.list[msg.ConversationID], msg)
	return msg, nil
}

func (m *memMsgStore) List(_ context.Context, conversationID string, beforeSeq int64, limit int) ([]domain.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.Message
	for _, msg := range m.list[conversationID] {
		if beforeSeq <= 0 || msg.Seq < beforeSeq {
			out = append(out, msg)
		}
	}
	if limit > 0 && len(out) > limit {
		out = out[len(out)-limit:]
	}
	return out, nil
}

func (m *memMsgStore) Search(_ context.Context, conversationID, query string, beforeSeq int64, limit int) ([]domain.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.Message
	q := strings.ToLower(query)
	for _, msg := range m.list[conversationID] {
		if msg.RecalledAt != nil {
			continue
		}
		if beforeSeq > 0 && msg.Seq >= beforeSeq {
			continue
		}
		if strings.Contains(strings.ToLower(msg.Body), q) {
			out = append(out, msg)
		}
	}
	// newest first
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (m *memMsgStore) ListAround(_ context.Context, conversationID string, aroundSeq int64, window int) ([]domain.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var out []domain.Message
	for _, msg := range m.list[conversationID] {
		if msg.Seq >= aroundSeq-int64(window) && msg.Seq <= aroundSeq+int64(window) {
			out = append(out, msg)
		}
	}
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if out[j].Seq < out[i].Seq {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out, nil
}

func (m *memMsgStore) FindByID(_ context.Context, id string) (domain.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	msg, ok := m.byID[id]
	if !ok {
		return domain.Message{}, apperr.NotFound("message not found")
	}
	return msg, nil
}

func (m *memMsgStore) FindByIDs(_ context.Context, ids []string) (map[string]domain.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := map[string]domain.Message{}
	for _, id := range ids {
		if msg, ok := m.byID[id]; ok {
			out[id] = msg
		}
	}
	return out, nil
}

func (m *memMsgStore) UpdateBody(_ context.Context, id, body string, editedAt time.Time) (domain.Message, error) {
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

func (m *memMsgStore) MarkRecalled(_ context.Context, id string, recalledAt time.Time) (domain.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	msg, ok := m.byID[id]
	if !ok {
		return domain.Message{}, apperr.NotFound("message not found")
	}
	msg.RecalledAt = &recalledAt
	m.byID[id] = msg
	return msg, nil
}

// memMsgMembers is a simple membership checker for handler tests.
type memMsgMembers struct {
	convMembers map[string]map[string]bool
}

func (m memMsgMembers) IsMember(_ context.Context, conversationID, userID string) (bool, error) {
	return m.convMembers[conversationID][userID], nil
}

func (m memMsgMembers) ListMemberIDs(_ context.Context, conversationID string) ([]string, error) {
	var out []string
	for uid, ok := range m.convMembers[conversationID] {
		if ok {
			out = append(out, uid)
		}
	}
	return out, nil
}

func messageHandlerHarness() (*MessageHandler, *memMsgStore) {
	store := newMemMsgStore()
	members := memMsgMembers{convMembers: map[string]map[string]bool{
		"c1": {"u1": true, "u2": true},
	}}
	svc := service.NewMessageService(store, members, nil)
	return &MessageHandler{Msg: svc}, store
}

func messageReq(path, userID string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req = req.WithContext(context.WithValue(req.Context(), authUserKey{}, userID))
	return req
}

func TestMessageSearchHandler(t *testing.T) {
	h, store := messageHandlerHarness()
	ctx := context.Background()
	seed := func(body string) {
		if _, err := store.Insert(ctx, domain.Message{ID: uuid.NewString(), ConversationID: "c1", SenderID: "u1", Body: body, ClientMsgID: body}); err != nil {
			t.Fatal(err)
		}
	}
	seed("hello world")
	seed("goodbye planet")
	seed("hello again")

	req := messageReq("/v1/conversations/c1/messages/search?q=hello", "u1")
	rr := httptest.NewRecorder()
	h.Search(rr, req, "c1")

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", rr.Code, rr.Body.String())
	}
	var out struct {
		Messages []messageDTO `json:"messages"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Messages) != 2 {
		t.Fatalf("want 2 hits, got %d", len(out.Messages))
	}
}

func TestMessageSearchHandlerBlankQuery(t *testing.T) {
	h, _ := messageHandlerHarness()
	req := messageReq("/v1/conversations/c1/messages/search?q=%20%20%20", "u1")
	rr := httptest.NewRecorder()
	h.Search(rr, req, "c1")
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}

func TestMessageSearchHandlerNonMember(t *testing.T) {
	h, _ := messageHandlerHarness()
	req := messageReq("/v1/conversations/c1/messages/search?q=hello", "ghost")
	rr := httptest.NewRecorder()
	h.Search(rr, req, "c1")
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rr.Code)
	}
}

func TestMessageListAroundHandler(t *testing.T) {
	h, store := messageHandlerHarness()
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		if _, err := store.Insert(ctx, domain.Message{ID: uuid.NewString(), ConversationID: "c1", SenderID: "u1", Body: "m" + strconv.Itoa(i), ClientMsgID: strconv.Itoa(i)}); err != nil {
			t.Fatal(err)
		}
	}
	req := messageReq("/v1/conversations/c1/messages?around_seq=3&limit=2", "u1")
	rr := httptest.NewRecorder()
	h.List(rr, req, "c1")
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body %s", rr.Code, rr.Body.String())
	}
	var out struct {
		Messages []messageDTO `json:"messages"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Messages) == 0 {
		t.Fatal("want around window, got none")
	}
	// ascending
	for i := 1; i < len(out.Messages); i++ {
		if out.Messages[i].Seq < out.Messages[i-1].Seq {
			t.Fatalf("not ascending: %d after %d", out.Messages[i].Seq, out.Messages[i-1].Seq)
		}
	}
}

func TestMessageListAroundBeforeMutuallyExclusive(t *testing.T) {
	h, _ := messageHandlerHarness()
	req := messageReq("/v1/conversations/c1/messages?around_seq=3&before_seq=2", "u1")
	rr := httptest.NewRecorder()
	h.List(rr, req, "c1")
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", rr.Code)
	}
}
