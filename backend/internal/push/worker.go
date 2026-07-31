package push

import (
	"context"
	"log/slog"
	"sync"

	"easy-im/backend/internal/domain"
)

// SubscriberLister fetches a user's push subscriptions.
type SubscriberLister interface {
	ListByUser(ctx context.Context, userID string) ([]domain.PushSubscription, error)
}

// StaleSubscriptionRemover removes subscriptions the push provider reports as gone.
type StaleSubscriptionRemover interface {
	DeleteByEndpoints(ctx context.Context, endpoints []string) error
}

// MemberLister lists conversation member user ids.
type MemberLister interface {
	ListMemberIDs(ctx context.Context, conversationID string) ([]string, error)
}

// UserGetter resolves a user's display name for the notification title.
type UserGetter interface {
	FindByID(ctx context.Context, id string) (domain.User, error)
}

// Dispatcher delivers a payload to one subscription and reports the outcome.
type Dispatcher interface {
	Send(ctx context.Context, endpoint, p256dh, auth string, payload []byte) SendResult
}

// PresenceTracker holds the online set learned from bus presence events.
type PresenceTracker struct {
	mu    sync.RWMutex
	users map[string]bool
}

// NewPresenceTracker returns an empty tracker.
func NewPresenceTracker() *PresenceTracker {
	return &PresenceTracker{users: map[string]bool{}}
}

// Set records a presence observation.
func (t *PresenceTracker) Set(userID string, online bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if online {
		t.users[userID] = true
	} else {
		delete(t.users, userID)
	}
}

// IsOnline reports whether the user currently has a live connection.
func (t *PresenceTracker) IsOnline(userID string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.users[userID]
}

// OfflineHandler consumes im.messages records and queues notifications for
// offline members through the shared aggregator (which the Flusher drains).
type OfflineHandler struct {
	members MemberLister
	users   UserGetter
	tracker *PresenceTracker
	agg     *Aggregator
	log     *slog.Logger
}

// NewOfflineHandler wires an offline handler onto the given aggregator.
func NewOfflineHandler(members MemberLister, users UserGetter, tracker *PresenceTracker, agg *Aggregator, log *slog.Logger) *OfflineHandler {
	return &OfflineHandler{members: members, users: users, tracker: tracker, agg: agg, log: log}
}

// HandleMessage is called for each consumed im.messages record.
func (h *OfflineHandler) HandleMessage(ctx context.Context, m domain.Message) {
	memberIDs, err := h.members.ListMemberIDs(ctx, m.ConversationID)
	if err != nil {
		h.log.Warn("list members failed", "conversation_id", m.ConversationID, "error", err)
		return
	}
	senderName := "New message"
	if u, err := h.users.FindByID(ctx, m.SenderID); err == nil && u.DisplayName != "" {
		senderName = u.DisplayName
	} else if err != nil {
		h.log.Warn("resolve sender failed", "sender_id", m.SenderID, "error", err)
	}
	for _, uid := range memberIDs {
		if uid == m.SenderID {
			continue // never push the sender's own message
		}
		if h.tracker.IsOnline(uid) {
			continue // online members get the realtime path
		}
		h.agg.Add(m.ConversationID, senderName, m.Body)
	}
}
