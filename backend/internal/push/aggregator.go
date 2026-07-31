package push

import (
	"sync"
	"time"
)

// PendingNotification is one conversation's aggregated notification state
// waiting for the window to elapse.
type PendingNotification struct {
	ConversationID string
	SenderName     string
	Preview        string
	Count          int
}

// Aggregator buffers per-conversation notifications and flushes them in a
// time window so several offline messages in the same conversation collapse
// into a single system notification ("N new messages").
type Aggregator struct {
	window  time.Duration
	mu      sync.Mutex
	pending map[string]*PendingNotification
	timers  map[string]*time.Timer
	flushFn OnFlush
}

// OnFlush is called with the aggregated notification for a conversation when
// its window elapses. Implementations must be safe to call concurrently.
type OnFlush func(p PendingNotification)

// NewAggregator returns an aggregator with the given window.
func NewAggregator(window time.Duration, flush OnFlush) *Aggregator {
	if window <= 0 {
		window = 2 * time.Second
	}
	if flush == nil {
		flush = func(PendingNotification) {}
	}
	return &Aggregator{
		window:  window,
		pending: map[string]*PendingNotification{},
		timers:  map[string]*time.Timer{},
		flushFn: flush,
	}
}

// Add queues a new message into its conversation's bucket. Subsequent messages
// in the same conversation within the window extend the bucket; the sender
// name and preview are those of the latest message, and the count increments.
func (a *Aggregator) Add(conversationID, senderName, preview string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	p, ok := a.pending[conversationID]
	if !ok {
		p = &PendingNotification{ConversationID: conversationID}
		a.pending[conversationID] = p
		a.timers[conversationID] = time.AfterFunc(a.window, func() {
			a.flush(conversationID)
		})
	}
	p.SenderName = senderName
	p.Preview = preview
	p.Count++
}

func (a *Aggregator) flush(conversationID string) {
	a.mu.Lock()
	p, ok := a.pending[conversationID]
	if ok {
		delete(a.pending, conversationID)
	}
	if t, ok := a.timers[conversationID]; ok {
		t.Stop()
		delete(a.timers, conversationID)
	}
	a.mu.Unlock()
	if ok {
		a.flushFn(*p)
	}
}

// Stop cancels all pending timers (e.g. on worker shutdown).
func (a *Aggregator) Stop() {
	a.mu.Lock()
	defer a.mu.Unlock()
	for id, t := range a.timers {
		t.Stop()
		delete(a.timers, id)
	}
	a.pending = map[string]*PendingNotification{}
}
