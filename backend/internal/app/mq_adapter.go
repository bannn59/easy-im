package app

import (
	"context"

	"easy-im/backend/internal/domain"
	"easy-im/backend/internal/mq"
)

// messageEventAdapter adapts the bus Producer to the service's narrow
// MessageEventPublisher interface. nodeID tags every event with its origin so
// the per-node realtime fanout consumer can skip events it produced itself
// (avoiding double delivery on the local node).
type messageEventAdapter struct {
	producer mq.Producer
	nodeID   string
}

func (a *messageEventAdapter) PublishMessageCreated(ctx context.Context, m domain.Message) error {
	if a == nil || a.producer == nil {
		return nil
	}
	ev := mq.NewMessageEvent(m)
	ev.Origin = a.nodeID
	return a.producer.Publish(ctx, mq.TopicMessages, m.ConversationID, ev)
}

func (a *messageEventAdapter) PublishMessageEdited(ctx context.Context, m domain.Message) error {
	if a == nil || a.producer == nil || m.EditedAt == nil {
		return nil
	}
	return a.producer.Publish(ctx, mq.TopicMessages, m.ConversationID, mq.NewEditedEvent(m, a.nodeID))
}

func (a *messageEventAdapter) PublishMessageRecalled(ctx context.Context, m domain.Message) error {
	if a == nil || a.producer == nil || m.RecalledAt == nil {
		return nil
	}
	return a.producer.Publish(ctx, mq.TopicMessages, m.ConversationID, mq.NewRecalledEvent(m, a.nodeID))
}

func (a *messageEventAdapter) PublishMessageRead(ctx context.Context, conversationID, userID string, lastReadSeq int64) error {
	if a == nil || a.producer == nil {
		return nil
	}
	return a.producer.Publish(ctx, mq.TopicMessages, conversationID, mq.NewReadEvent(conversationID, userID, lastReadSeq, a.nodeID))
}
