package push

import (
	"sync"
	"testing"
	"time"
)

func TestAggregatorCollapsesSameConversation(t *testing.T) {
	var (
		mu    sync.Mutex
		flush []PendingNotification
	)
	a := NewAggregator(30*time.Millisecond, func(p PendingNotification) {
		mu.Lock()
		flush = append(flush, p)
		mu.Unlock()
	})

	a.Add("conv-1", "alice", "hello")
	a.Add("conv-1", "alice", "hi again")
	a.Add("conv-1", "alice", "third")

	waitForFlush(t, &mu, func() int { return len(flush) }, 1)

	mu.Lock()
	defer mu.Unlock()
	if len(flush) != 1 {
		t.Fatalf("expected 1 flush, got %d", len(flush))
	}
	got := flush[0]
	if got.ConversationID != "conv-1" {
		t.Fatalf("conversation = %q", got.ConversationID)
	}
	if got.Count != 3 {
		t.Fatalf("count = %d, want 3", got.Count)
	}
	if got.Preview != "third" {
		t.Fatalf("preview = %q, want latest 'third'", got.Preview)
	}
}

func TestAggregatorSeparateConversations(t *testing.T) {
	var (
		mu    sync.Mutex
		flush []PendingNotification
	)
	a := NewAggregator(20*time.Millisecond, func(p PendingNotification) {
		mu.Lock()
		flush = append(flush, p)
		mu.Unlock()
	})

	a.Add("conv-1", "alice", "a1")
	a.Add("conv-2", "bob", "b1")

	waitForFlush(t, &mu, func() int { return len(flush) }, 2)

	mu.Lock()
	defer mu.Unlock()
	if len(flush) != 2 {
		t.Fatalf("expected 2 flushes, got %d", len(flush))
	}
	for _, p := range flush {
		if p.Count != 1 {
			t.Fatalf("conversation %s count = %d, want 1", p.ConversationID, p.Count)
		}
	}
}

func TestAggregatorStopCancels(t *testing.T) {
	var flush []PendingNotification
	a := NewAggregator(10*time.Millisecond, func(p PendingNotification) {
		flush = append(flush, p)
	})
	a.Add("conv-1", "alice", "x")
	a.Stop()
	time.Sleep(30 * time.Millisecond)
	if len(flush) != 0 {
		t.Fatalf("expected no flush after Stop, got %d", len(flush))
	}
}

// waitForFlush polls until cond() meets want or the timeout elapses.
func waitForFlush(t *testing.T, mu *sync.Mutex, cond func() int, want int) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		n := cond()
		mu.Unlock()
		if n >= want {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %d flushes", want)
}
