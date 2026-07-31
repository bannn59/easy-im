package app

import (
	"context"

	"easy-im/backend/internal/domain"
	"easy-im/backend/internal/mq"
)

// messageEventAdapter adapts the bus Producer to the service's narrow
// MessageEventPublisher interface (published per stored message).
type messageEventAdapter struct {
	producer mq.Producer
}

func (a *messageEventAdapter) PublishMessageCreated(ctx context.Context, m domain.Message) error {
	if a == nil || a.producer == nil {
		return nil
	}
	return a.producer.Publish(ctx, mq.TopicMessages, m.ConversationID, mq.NewMessageEvent(m))
}
