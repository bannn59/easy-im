package hub

import (
	"encoding/json"
	"sync"

	"github.com/gorilla/websocket"
)

// Event is a server-pushed frame.
type Event struct {
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
}

func New() *Hub {
	return &Hub{clients: map[string]map[*Client]struct{}{}}
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

func (c *Client) WritePump() {
	for msg := range c.Send {
		if err := c.Conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}

// ReadPump reads until disconnect.
func (c *Client) ReadPump(onClose func()) {
	defer onClose()
	for {
		if _, _, err := c.Conn.ReadMessage(); err != nil {
			return
		}
	}
}
