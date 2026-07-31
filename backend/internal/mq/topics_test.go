package mq

import (
	"encoding/json"
	"testing"
	"time"

	"easy-im/backend/internal/domain"
)

func mustMsg(t *testing.T, createdAt time.Time) domain.Message {
	t.Helper()
	return domain.Message{
		ID:             "m1",
		ConversationID: "c1",
		SenderID:       "u1",
		Body:           "hi",
		CreatedAt:      createdAt,
	}
}

func TestMessageEventDefaultTypeIsCreated(t *testing.T) {
	// Older records predating the type field must decode as "created".
	raw := `{"id":"m1","conversation_id":"c1","sender_id":"u1","body":"hi"}`
	var ev MessageEvent
	if err := json.Unmarshal([]byte(raw), &ev); err != nil {
		t.Fatal(err)
	}
	if ev.EventType() != MessageCreated {
		t.Fatalf("want created, got %q", ev.EventType())
	}
}

func TestNewMessageEventCarriesOrigin(t *testing.T) {
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	ev := NewMessageEvent(mustMsg(t, now))
	ev.Origin = "host:123"
	b, err := json.Marshal(ev)
	if err != nil {
		t.Fatal(err)
	}
	// Worker decodes with the old field set intact.
	var decoded MessageEvent
	if err := json.Unmarshal(b, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.ID != "m1" || decoded.ConversationID != "c1" || decoded.SenderID != "u1" || decoded.Body != "hi" {
		t.Fatalf("decode mismatch: %+v", decoded)
	}
	if decoded.Origin != "host:123" {
		t.Fatalf("origin not carried: %+v", decoded)
	}
	if decoded.EventType() != MessageCreated {
		t.Fatalf("want created, got %q", decoded.EventType())
	}
}

func TestEditedRecalledRoundTrip(t *testing.T) {
	now := time.Date(2026, 7, 31, 12, 0, 0, 0, time.UTC)
	edited := now.Add(time.Minute)
	m := mustMsg(t, now)
	m.EditedAt = &edited

	ee := NewEditedEvent(m, "node1")
	b, _ := json.Marshal(ee)
	var dec MessageEvent
	if err := json.Unmarshal(b, &dec); err != nil {
		t.Fatal(err)
	}
	if dec.EventType() != MessageEdited {
		t.Fatalf("want edited, got %q", dec.EventType())
	}
	dm := dec.ToDomain()
	if dm.EditedAt == nil || !dm.EditedAt.Equal(edited) {
		t.Fatalf("edited_at round trip failed: %+v", dm.EditedAt)
	}

	recalled := now.Add(2 * time.Minute)
	m.RecalledAt = &recalled
	re := NewRecalledEvent(m, "node1")
	b2, _ := json.Marshal(re)
	var dec2 MessageEvent
	if err := json.Unmarshal(b2, &dec2); err != nil {
		t.Fatal(err)
	}
	if dec2.EventType() != MessageRecalled {
		t.Fatalf("want recalled, got %q", dec2.EventType())
	}
	dm2 := dec2.ToDomain()
	if dm2.RecalledAt == nil || !dm2.RecalledAt.Equal(recalled) {
		t.Fatalf("recalled_at round trip failed: %+v", dm2.RecalledAt)
	}
}

func TestReadEventRoundTrip(t *testing.T) {
	re := NewReadEvent("c1", "u2", 42, "node1")
	b, _ := json.Marshal(re)
	var dec MessageEvent
	if err := json.Unmarshal(b, &dec); err != nil {
		t.Fatal(err)
	}
	if dec.EventType() != MessageRead {
		t.Fatalf("want read, got %q", dec.EventType())
	}
	if dec.ReadByUserID != "u2" || dec.LastReadSeq != 42 || dec.ConversationID != "c1" {
		t.Fatalf("read event mismatch: %+v", dec)
	}
}
