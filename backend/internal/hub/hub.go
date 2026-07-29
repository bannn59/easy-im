package hub

import (
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Event is a server-pushed frame.
type Event struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// InboundFrame is a client-to-server frame.
type InboundFrame struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// Client is one websocket connection for a user.
type Client struct {
	UserID string
	Conn   *websocket.Conn
	Send   chan []byte
}

// Hub tracks online connections by user id (multi-device supported).
type Hub struct {
	mu      sync.RWMutex
	clients map[string]map[*Client]struct{}

	// FrameHandler is called for each valid inbound frame. Set by the WS handler.
	FrameHandler func(userID string, frame InboundFrame)

	// Typing timers: key = "conversationID:userID".
	typingMu     sync.Mutex
	typingTimers map[string]*time.Timer
}

func New() *Hub {
	return &Hub{
		clients:      map[string]map[*Client]struct{}{},
		typingTimers: map[string]*time.Timer{},
	}
}

func (h *Hub) Register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.clients[c.UserID] == nil {
		h.clients[c.UserID] = map[*Client]struct{}{}
	}
	h.clients[c.UserID][c] = struct{}{}
}

func (h *Hub) Unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if set, ok := h.clients[c.UserID]; ok {
		if _, exists := set[c]; exists {
			delete(set, c)
			close(c.Send)
		}
		if len(set) == 0 {
			delete(h.clients, c.UserID)
		}
	}
	_ = c.Conn.Close()
}

// PublishToUsers sends event JSON to all connections of the given users.
func (h *Hub) PublishToUsers(userIDs []string, event Event) {
	data, err := json.Marshal(event)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	seen := map[*Client]struct{}{}
	for _, uid := range userIDs {
		for c := range h.clients[uid] {
			if _, ok := seen[c]; ok {
				continue
			}
			seen[c] = struct{}{}
			select {
			case c.Send <- data:
			default:
			}
		}
	}
}

// BroadcastToConversation sends event to all given members except exceptUserID.
func (h *Hub) BroadcastToConversation(memberIDs []string, exceptUserID string, event Event) {
	var targets []string
	for _, id := range memberIDs {
		if id != exceptUserID {
			targets = append(targets, id)
		}
	}
	if len(targets) > 0 {
		h.PublishToUsers(targets, event)
	}
}

// SetTyping arms (or resets) a server-side typing timeout for the given
// conversation+user. onExpired is called when the timeout fires.
func (h *Hub) SetTyping(conversationID, userID string, onExpired func()) {
	key := conversationID + ":" + userID
	h.typingMu.Lock()
	defer h.typingMu.Unlock()
	if t, ok := h.typingTimers[key]; ok {
		t.Stop()
	}
	h.typingTimers[key] = time.AfterFunc(3*time.Second, func() {
		h.ClearTyping(conversationID, userID)
		onExpired()
	})
}

// ClearTyping cancels any active typing timer for the given conversation+user.
func (h *Hub) ClearTyping(conversationID, userID string) {
	key := conversationID + ":" + userID
	h.typingMu.Lock()
	defer h.typingMu.Unlock()
	if t, ok := h.typingTimers[key]; ok {
		t.Stop()
		delete(h.typingTimers, key)
	}
}

func (c *Client) WritePump() {
	for msg := range c.Send {
		if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}

// ReadPump reads inbound frames until disconnect, dispatching via FrameHandler.
func (h *Hub) ReadPump(c *Client, onClose func()) {
	defer onClose()
	for {
		_, raw, err := c.Conn.ReadMessage()
		if err != nil {
			return
		}
		var f InboundFrame
		if err := json.Unmarshal(raw, &f); err != nil {
			continue // ignore malformed frames
		}
		if f.Type == "" {
			continue
		}
		if h.FrameHandler != nil {
			h.FrameHandler(c.UserID, f)
		}
	}
}
