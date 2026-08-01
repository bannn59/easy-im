# Logging Guidelines

> Structured logging for easy-im backend processes.

---

## Bootstrap status

Assumed library: Go `log/slog` (stdlib) or a thin wrapper around it. If the team standardizes on Zap/Zerolog later, keep field names stable.

---

## Goals

- Correlate API requests, WS connections, and MQ consumers.
- Debug delivery issues without printing full chat contents in default prod config.
- Keep PII and secrets out of logs.

---

## Log levels

| Level | Use |
|-------|-----|
| Debug | Frame-level traces, SQL timing in dev |
| Info | Process start, listen addr, successful subsystem connect, notable business milestones (conn established counts are sampled) |
| Warn | Retries, slow queries, reconnect storms, degraded dependency |
| Error | Unexpected failures, handler 500s, consumer handler failures |

Fatal/exit logging only in `cmd` during startup misconfig.

---

## Required fields

Attach when available:

| Field | Meaning |
|-------|---------|
| `service` | `api` / `gateway` / `worker` |
| `request_id` | HTTP request id |
| `conn_id` | WebSocket connection id |
| `user_id` | Authenticated user (if known) |
| `device_id` | Device id for multi-device |
| `conversation_id` | When in message paths |
| `message_id` / `client_msg_id` | Delivery debugging |
| `event_id` | MQ event id |
| `error` | error string / chain for error logs |

Use consistent key names across processes so log backends can join on them.

---

## What to log

- Auth failures (without raw tokens).
- Message accept/reject at service boundary (ids, not full body by default).
- Gateway connect/disconnect with reason codes (sample if extremely high volume).
- MQ publish/consume failures.
- Migration and startup configuration (redact secrets).

---

## What NOT to log

- Access tokens, refresh tokens, passwords, WS tickets.
- Full message bodies / media URLs with auth query strings in **info** logs (debug-only and guarded).
- Entire Redis payloads or large JSON frames.
- Other users’ PII beyond what operators are allowed to see in your threat model.

---

## Request / connection correlation

1. API middleware generates or accepts `X-Request-ID`, puts it in `context`, adds to log handler and error responses.
2. Gateway assigns `conn_id` on upgrade; include in all subsequent logs for that conn.
3. When a WS action triggers HTTP-like use-cases, pass a `request_id` from the client frame if present.

---

## Sampling & volume

Gateway and fan-out paths can be extremely chatty. Rules:

- Heartbeats: debug only, or aggregate metrics instead of per-beat info logs.
- Prefer **metrics** (Prometheus) for rates; logs for exceptions.
- Cap error log storms with simple rate limiting if a dependency fails hard.

---

## Anti-patterns

- `fmt.Println` / bare `log.Printf` in library packages.
- Logging `err` without fields that identify the conversation/user/request.
- Different field names per process (`userId` vs `user_id`).
- Info-logging every outbound push frame in production.

---

## Metrics (Prometheus)

easy-im processes expose a Prometheus `/metrics` endpoint on a **dedicated
port** (`METRICS_ADDR`, api default `:9090`, worker default `:9091`) so
business routes are never polluted. Empty `METRICS_ADDR` disables the server
(process behavior is unchanged — all instrumentation is nil-safe).

### Metric naming

Prefix every metric with `easyim_` + subsystem, e.g.:

| Subsystem | Examples |
|-----------|----------|
| `http` | `easyim_http_requests_total{service,method,path,status}`, `easyim_http_request_duration_seconds{method,path}` |
| `ws` | `easyim_ws_online_conns`, `easyim_ws_online_users`, `easyim_ws_connections_total{service}` |
| `messages` | `easyim_messages_sent_total` |
| `fanout` | `easyim_fanout_events_total{event_type}`, `easyim_fanout_skipped_total{reason}` |
| `kafka` | `easyim_kafka_publish_total{topic,result}`, `easyim_kafka_consume_total{topic,group}` |
| `push` | `easyim_push_sent_total{result}`, `easyim_push_aggregated_total` |

### Label conventions

- Reuse the structured-logging field names (`service`, `event_type`, `result`,
  `topic`, `group`) so logs and metrics join cleanly.
- Normalize dynamic HTTP path segments to `{id}` (UUIDs) to keep label
  cardinality bounded.
- Do not put PII, tokens, or full message bodies in labels or metric values.

### Where metrics live

- `internal/metrics` owns the server + registry and every metric declaration.
- Instrumentation calls are sprinkled at the layer boundaries (handler
  middleware, hub, mq, service, push); they must be nil-safe and never block
  the business path.

### What to instrument

- Rates and volumes (HTTP, WS connections, message send, Kafka publish/consume,
  push) as counters.
- Latency (HTTP request duration) as histograms.
- Current state (WS online conns/users) as gauges.

### What NOT to instrument

- Per-heartbeat counters (log at debug instead).
- Anything requiring request-scoped state beyond method/path/status labels.

---

## Verification

- Grep for `fmt.Println` / `log.Print` in `internal/` once code exists.
- Confirm error responses include `request_id` matching logs.
