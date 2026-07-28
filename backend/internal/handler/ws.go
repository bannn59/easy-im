package handler

import (
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
	Auth *service.AuthService
	Hub  *hub.Hub
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
	client.ReadPump(func() { h.Hub.Unregister(client) })
}
