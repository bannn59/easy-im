package push

import (
	"context"
	"log/slog"
	"time"
)

// Flusher delivers aggregated notifications to offline conversation members'
// subscriptions and prunes stale ones. It owns the Aggregator's flush hook.
type Flusher struct {
	members MemberLister
	tracker *PresenceTracker
	subs    SubscriberLister
	remover StaleSubscriptionRemover
	sender  Dispatcher
	log     *slog.Logger
}

// NewFlusher builds a Flusher with an Aggregator whose flush delivers pushes.
func NewFlusher(members MemberLister, tracker *PresenceTracker, subs SubscriberLister, remover StaleSubscriptionRemover, sender Dispatcher, window time.Duration, log *slog.Logger) (*Flusher, *Aggregator) {
	f := &Flusher{members: members, tracker: tracker, subs: subs, remover: remover, sender: sender, log: log}
	agg := NewAggregator(window, f.flush)
	return f, agg
}

// flush sends one aggregated notification to every subscription of the
// conversation's offline members, then prunes gone subscriptions. Recipients
// are re-derived at flush time so membership and online state are current.
func (f *Flusher) flush(p PendingNotification) {
	ctx := context.Background()
	payload, err := MarshalPayload(NotificationPayload{
		Type:           "chat_message",
		Title:          p.SenderName,
		Body:           p.Preview,
		ConversationID: p.ConversationID,
		Tag:            p.ConversationID,
		Count:          p.Count,
	})
	if err != nil {
		f.log.Warn("marshal push payload failed", "conversation_id", p.ConversationID, "error", err)
		return
	}

	memberIDs, err := f.members.ListMemberIDs(ctx, p.ConversationID)
	if err != nil {
		f.log.Warn("list members failed", "conversation_id", p.ConversationID, "error", err)
		return
	}

	var stale []string
	for _, uid := range memberIDs {
		if f.tracker.IsOnline(uid) {
			continue
		}
		subs, err := f.subs.ListByUser(ctx, uid)
		if err != nil {
			f.log.Warn("list subscriptions failed", "user_id", uid, "error", err)
			continue
		}
		for _, sub := range subs {
			res := f.sender.Send(ctx, sub.Endpoint, sub.P256DH, sub.Auth, payload)
			if res.Gone {
				stale = append(stale, sub.Endpoint)
			} else if res.Err != nil {
				f.log.Warn("push send failed", "endpoint", sub.Endpoint, "error", res.Err)
			}
		}
	}
	if len(stale) > 0 && f.remover != nil {
		if err := f.remover.DeleteByEndpoints(ctx, stale); err != nil {
			f.log.Warn("delete stale subscriptions failed", "error", err)
		}
	}
}
