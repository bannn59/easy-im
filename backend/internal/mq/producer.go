package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"

	"easy-im/backend/internal/metrics"
)

// Producer publishes bus events to Kafka. Implementations must be safe to call
// from message paths (non-blocking on broker outage).
type Producer interface {
	// Publish enqueues one JSON-encoded event to topic with the given key.
	// Returns an error only when the event cannot even be queued; broker
	// failures are reported via the optional ErrHandler, not here.
	Publish(ctx context.Context, topic, key string, v any) error
	Close()
}

// KafkaProducer is the franz-go-backed Producer.
type KafkaProducer struct {
	client *kgo.Client
	onErr  func(error)
}

// ProducerOpts configures KafkaProducer.
type ProducerOpts struct {
	// Brokers is the broker list to seed.
	Brokers []string
	// ClientID identifies this producer in broker logs.
	ClientID string
	// OnError, if set, receives asynchronous publish errors for observability.
	OnError func(err error)
}

// NewKafkaProducer connects and returns a Producer.
func NewKafkaProducer(opts ProducerOpts) (Producer, error) {
	client, err := kgo.NewClient(
		kgo.SeedBrokers(opts.Brokers...),
		kgo.ClientID(opts.ClientID),
		// Buffered, non-blocking produce: records queue internally and a
		// background goroutine flushes them, so message send never stalls on
		// a slow broker. Delivery errors surface through the callback.
		kgo.MaxBufferedRecords(1000),
		kgo.MaxBufferedBytes(16<<20), // 16 MiB
		kgo.ProducerLinger(10*time.Millisecond),
	)
	if err != nil {
		return nil, fmt.Errorf("new kafka client: %w", err)
	}
	p := &KafkaProducer{client: client, onErr: opts.OnError}
	return p, nil
}

// ProduceSync issues a synchronous produce so callers (e.g. tests) can observe
// the outcome directly.
func (p *KafkaProducer) ProduceSync(ctx context.Context, topic, key string, v any) error {
	start := time.Now()
	val, err := json.Marshal(v)
	if err != nil {
		metrics.KafkaPublishTotal.WithLabelValues(topic, "marshal_err").Inc()
		return fmt.Errorf("marshal event: %w", err)
	}
	rec := &kgo.Record{Topic: topic, Key: []byte(key), Value: val}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := p.client.ProduceSync(ctx, rec).FirstErr(); err != nil {
		metrics.KafkaPublishTotal.WithLabelValues(topic, "error").Inc()
		return err
	}
	metrics.KafkaPublishTotal.WithLabelValues(topic, "ok").Inc()
	metrics.KafkaPublishDuration.WithLabelValues(topic).Observe(time.Since(start).Seconds())
	return nil
}

// Publish enqueues an event for async delivery. The passed ctx is used for
// marshalling only; the produce itself is backgrounded (franz-go delivers
// asynchronously), so it must not be tied to a request-scoped context that
// may be cancelled after the HTTP handler returns.
func (p *KafkaProducer) Publish(ctx context.Context, topic, key string, v any) error {
	val, err := json.Marshal(v)
	if err != nil {
		metrics.KafkaPublishTotal.WithLabelValues(topic, "marshal_err").Inc()
		return fmt.Errorf("marshal event: %w", err)
	}
	rec := &kgo.Record{Topic: topic, Key: []byte(key), Value: val}
	p.client.Produce(context.Background(), rec, func(r *kgo.Record, err error) {
		if err != nil {
			metrics.KafkaPublishTotal.WithLabelValues(topic, "error").Inc()
			if p.onErr != nil {
				p.onErr(fmt.Errorf("publish %s: %w", topic, err))
			}
		} else {
			metrics.KafkaPublishTotal.WithLabelValues(topic, "ok").Inc()
		}
	})
	return nil
}

// Close flushes and closes the client.
func (p *KafkaProducer) Close() {
	if p.client != nil {
		p.client.Close()
	}
}

// ---- nil-safe producer for processes without Kafka ----

type noopProducer struct{}

// NoopProducer discards every event; used when Kafka is not configured so
// message send never blocks on a missing bus.
var NoopProducer Producer = noopProducer{}

func (noopProducer) Publish(_ context.Context, _, _ string, _ any) error { return nil }
func (noopProducer) Close()                                              {}
