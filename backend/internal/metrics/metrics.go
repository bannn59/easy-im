package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// All easy-im metrics are registered on the default registry so a single
// promhttp.Handler exposes every process. Labels reuse the structured-logging
// field names from .trellis/spec/backend/logging-guidelines.md (service,
// event_type, result, ...) so logs and metrics join cleanly.

// HTTP request metrics, recorded by the handler middleware.
var (
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: "http",
			Name:      "requests_total",
			Help:      "Total HTTP requests handled, by service/method/path/status.",
		},
		[]string{"service", "method", "path", "status"},
	)
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: Namespace,
			Subsystem: "http",
			Name:      "request_duration_seconds",
			Help:      "HTTP request duration in seconds, by method/path.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "path"},
	)
)

// WebSocket metrics, recorded by the hub.
var (
	WSOnlineConns = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: "ws",
			Name:      "online_conns",
			Help:      "Current number of live WebSocket connections on this node.",
		},
	)
	WSOnlineUsers = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: Namespace,
			Subsystem: "ws",
			Name:      "online_users",
			Help:      "Current number of users with at least one live connection on this node.",
		},
	)
	WSConnectionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: "ws",
			Name:      "connections_total",
			Help:      "Total WebSocket connections opened, by service.",
		},
		[]string{"service"},
	)
)

// Message metrics, recorded by the message service and fanout consumer.
var (
	MessagesSentTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: "messages",
			Name:      "sent_total",
			Help:      "Total messages durably stored and broadcast.",
		},
	)
	FanoutEventsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: "fanout",
			Name:      "events_total",
			Help:      "Total bus message events consumed by the fanout consumer, by event type.",
		},
		[]string{"event_type"},
	)
	FanoutSkippedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: "fanout",
			Name:      "skipped_total",
			Help:      "Total bus events skipped by the fanout consumer, by reason.",
		},
		[]string{"reason"},
	)
)

// Kafka metrics, recorded by the producer and consumer.
var (
	KafkaPublishTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: "kafka",
			Name:      "publish_total",
			Help:      "Total Kafka publish attempts, by topic and result.",
		},
		[]string{"topic", "result"},
	)
	KafkaPublishDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: Namespace,
			Subsystem: "kafka",
			Name:      "publish_duration_seconds",
			Help:      "Kafka publish duration in seconds, by topic.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"topic"},
	)
	KafkaConsumeTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: "kafka",
			Name:      "consume_total",
			Help:      "Total Kafka records consumed, by topic and consumer group.",
		},
		[]string{"topic", "group"},
	)
	KafkaConsumeErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: "kafka",
			Name:      "consume_errors_total",
			Help:      "Total Kafka records whose handler failed, by topic and consumer group.",
		},
		[]string{"topic", "group"},
	)
)

// Push metrics, recorded by the worker.
var (
	PushSentTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: "push",
			Name:      "sent_total",
			Help:      "Total Web Push deliveries attempted, by result.",
		},
		[]string{"result"},
	)
	PushAggregatedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: "push",
			Name:      "aggregated_total",
			Help:      "Total offline notifications queued for push.",
		},
	)
	PushAggregateBatchesTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Namespace: Namespace,
			Subsystem: "push",
			Name:      "aggregate_batches_total",
			Help:      "Total aggregator flush batches delivered.",
		},
	)
)
