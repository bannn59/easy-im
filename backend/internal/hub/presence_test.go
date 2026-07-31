package hub

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/gorilla/websocket"
)

// startTestWSServer runs a gorilla echo server that keeps connections open.
func startTestWSServer(t *testing.T) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		up := websocket.Upgrader{CheckOrigin: func(*http.Request) bool { return true }}
		c, err := up.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer c.Close()
		for {
			if _, _, err := c.ReadMessage(); err != nil {
				return
			}
		}
	}))
	t.Cleanup(srv.Close)
	return "ws" + strings.TrimPrefix(srv.URL, "http")
}

func newTestClient(t *testing.T, wsURL, userID string) *Client {
	t.Helper()
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return &Client{UserID: userID, Conn: conn, Send: make(chan []byte, 16)}
}

func TestPresenceBroadcastOnTransition(t *testing.T) {
	wsURL := startTestWSServer(t)
	h := New()
	var mu sync.Mutex
	var online bool
	var called int
	h.PresenceBroadcaster = func(userID string, o bool) {
		mu.Lock()
		defer mu.Unlock()
		online = o
		called++
	}

	// First connection → online transition (1 event).
	c1 := newTestClient(t, wsURL, "u1")
	h.Register(c1)
	if called != 1 {
		t.Fatalf("want 1 presence event after first connect, got %d", called)
	}

	// Second device for same user → no new event (still online).
	c2 := newTestClient(t, wsURL, "u1")
	h.Register(c2)
	if called != 1 {
		t.Fatalf("want still 1 presence event after second connect, got %d", called)
	}

	// Closing one of two connections → still online, no event.
	h.Unregister(c1)
	if called != 1 {
		t.Fatalf("want still 1 presence event after closing one of two, got %d", called)
	}

	// Closing the last connection → offline transition (2 events total).
	h.Unregister(c2)
	if called != 2 {
		t.Fatalf("want 2 presence events after closing all, got %d", called)
	}
	mu.Lock()
	lastOnline := online
	mu.Unlock()
	if lastOnline {
		t.Fatal("want offline after closing last connection")
	}

	// Reconnect → online again (3 events).
	c3 := newTestClient(t, wsURL, "u1")
	h.Register(c3)
	if called != 3 {
		t.Fatalf("want 3 presence events after reconnect, got %d", called)
	}
}

func TestIsOnline(t *testing.T) {
	wsURL := startTestWSServer(t)
	h := New()
	if h.IsOnline("u1") {
		t.Fatal("u1 should be offline initially")
	}
	c := newTestClient(t, wsURL, "u1")
	h.Register(c)
	if !h.IsOnline("u1") {
		t.Fatal("u1 should be online after register")
	}
	h.Unregister(c)
	if h.IsOnline("u1") {
		t.Fatal("u1 should be offline after unregister")
	}
}

func TestOnlineUserIDs(t *testing.T) {
	wsURL := startTestWSServer(t)
	h := New()
	h.Register(newTestClient(t, wsURL, "u1"))
	h.Register(newTestClient(t, wsURL, "u2"))
	ids := h.OnlineUserIDs()
	if len(ids) != 2 {
		t.Fatalf("want 2 online users, got %v", ids)
	}
}
