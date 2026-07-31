package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/gorilla/websocket"

	"easy-im/backend/internal/apperr"
	"easy-im/backend/internal/hub"
	"easy-im/backend/internal/service"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true }, // dev only; tighten for production
}

// WSHandler upgrades to websocket after JWT auth.
type WSHandler struct {
	Auth    *service.AuthService
	Hub     *hub.Hub
	Members service.MembershipChecker
	Friends *service.FriendService
	Log     *slog.Logger
}

func (h *WSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.Auth == nil || h.Hub == nil {
		WriteError(w, r, apperr.Unavailable("realtime not configured"))
		return
	}
	token := bearerToken(r)
	if token == "" {
		token = r.URL.Query().Get("token")
	}
	uid, err := h.Auth.ParseAccessToken(token)
	if err != nil {
		WriteError(w, r, err)
		return
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	client := &hub.Client{UserID: uid, Conn: conn, Send: make(chan []byte, 16)}
	h.Hub.Register(client)
	go client.WritePump()
	h.Hub.ReadPump(client, func() { h.Hub.Unregister(client) })
}

// HandleFrame dispatches inbound WebSocket frames by type.
func (h *WSHandler) HandleFrame(userID string, f hub.InboundFrame) {
	switch f.Type {
	case "typing.start", "typing.stop":
		h.handleTyping(userID, f)
	default:
		if h.Log != nil {
			h.Log.Warn("unknown ws frame type", "type", f.Type, "user_id", userID)
		}
	}
}

// broadcastPresence relays an online/offline transition to the user's friends.
func (h *WSHandler) broadcastPresence(userID string, online bool) {
	if h.Hub == nil || h.Friends == nil {
		return
	}
	ctx := context.Background()
	friendIDs, err := h.Friends.ListFriendIDs(ctx, userID)
	if err != nil || len(friendIDs) == 0 {
		return
	}
	payload, err := json.Marshal(map[string]any{
		"user_id": userID,
		"online":  online,
	})
	if err != nil {
		return
	}
	h.Hub.PublishToUsers(friendIDs, hub.Event{Type: "presence.changed", Payload: payload})
}

type typingPayload struct {
	ConversationID string `json:"conversation_id"`
}

func (h *WSHandler) handleTyping(userID string, f hub.InboundFrame) {
	if h.Members == nil || h.Hub == nil {
		return
	}
	var p typingPayload
	if err := json.Unmarshal(f.Payload, &p); err != nil || p.ConversationID == "" {
		return
	}
	ctx := context.Background()
	ok, err := h.Members.IsMember(ctx, p.ConversationID, userID)
	if err != nil || !ok {
		return
	}
	memberIDs, err := h.Members.ListMemberIDs(ctx, p.ConversationID)
	if err != nil {
		return
	}

	eventType := "typing.started"
	if f.Type == "typing.stop" {
		eventType = "typing.stopped"
	}

	payload, _ := json.Marshal(map[string]string{
		"conversation_id": p.ConversationID,
		"user_id":         userID,
	})
	h.Hub.BroadcastToConversation(memberIDs, userID, hub.Event{Type: eventType, Payload: payload})

	if f.Type == "typing.start" {
		convID := p.ConversationID
		h.Hub.SetTyping(convID, userID, func() {
			stopPayload, _ := json.Marshal(map[string]string{
				"conversation_id": convID,
				"user_id":         userID,
			})
			h.Hub.BroadcastToConversation(memberIDs, userID, hub.Event{Type: "typing.stopped", Payload: stopPayload})
		})
	} else {
		h.Hub.ClearTyping(p.ConversationID, userID)
	}
}
