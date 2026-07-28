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

## Verification

- Grep for `fmt.Println` / `log.Print` in `internal/` once code exists.
- Confirm error responses include `request_id` matching logs.
