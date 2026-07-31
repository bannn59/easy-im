package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

// ConsumerOpts configures a consumer group.
type ConsumerOpts struct {
	// Brokers is the broker list to seed.
	Brokers []string
	// Group is the consumer group id (offsets are stored per group).
	Group string
	// ClientID identifies this consumer in broker logs.
	ClientID string
	// Topics to subscribe to.
	Topics []string
	// StartAtEnd, when true, makes a brand-new group begin at the latest offset
	// instead of the earliest. Offsets already committed by the group are
	// always resumed regardless of this flag. Use for realtime fanout groups
	// where replaying historical records on first join is undesirable.
	StartAtEnd bool
	// Log receives consumer-level events.
	Log *slog.Logger
}

// Message is a single bus record delivered to a handler.
type Message struct {
	Topic string
	Key   string
	Value []byte
}

// Consumer runs a Kafka consumer group. Handlers receive records sequentially
// per partition; call Commit to persist progress (at-least-once).
type Consumer struct {
	client *kgo.Client
	topics []string
	log    *slog.Logger
}

// NewConsumer connects and returns a Consumer. Handlers are registered via
// Register before calling Run.
func NewConsumer(opts ConsumerOpts) (*Consumer, error) {
	consumeOpts := []kgo.Opt{
		kgo.SeedBrokers(opts.Brokers...),
		kgo.ClientID(opts.ClientID),
		kgo.ConsumerGroup(opts.Group),
		kgo.ConsumeTopics(opts.Topics...),
		// Commit offsets every 5s or 1k records, whichever comes first.
		kgo.DisableAutoCommit(),
	}
	if opts.StartAtEnd {
		consumeOpts = append(consumeOpts, kgo.ConsumeResetOffset(kgo.NewOffset().AtEnd()))
	}
	client, err := kgo.NewClient(consumeOpts...)
	if err != nil {
		return nil, fmt.Errorf("new kafka consumer: %w", err)
	}
	return &Consumer{client: client, topics: opts.Topics, log: opts.Log}, nil
}

// Run consumes records until ctx is cancelled, dispatching each to handler.
// Handlers run serially (one goroutine), preserving per-partition order.
func (c *Consumer) Run(ctx context.Context, handler func(ctx context.Context, msg Message) error) error {
	for {
		fetches := c.client.PollFetches(ctx)
		if err := fetches.Err(); err != nil {
			// Cancelled ctx is a clean shutdown.
			if ctx.Err() != nil {
				return nil
			}
			c.log.Warn("kafka poll error", "error", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}
		fetches.EachRecord(func(r *kgo.Record) {
			msg := Message{Topic: r.Topic, Key: string(r.Key), Value: r.Value}
			if err := handler(ctx, msg); err != nil {
				c.log.Error("kafka handler failed", "topic", r.Topic, "error", err)
			}
			// Advance the consumer-group offset after processing. Records
			// whose handler failed are skipped (at-least-once boundary; a
			// poison message stays uncommitted and is retried on restart).
			_ = c.client.CommitRecords(context.Background(), r)
		})
	}
}

// Close closes the client, flushing pending commits.
func (c *Consumer) Close() {
	if c.client != nil {
		c.client.Close()
	}
}

// DecodeInto unmarshals a message value into v.
func DecodeInto(msg Message, v any) error {
	if err := json.Unmarshal(msg.Value, v); err != nil {
		return fmt.Errorf("decode %s record: %w", msg.Topic, err)
	}
	return nil
}
