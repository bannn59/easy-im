# Error Handling

> How failures are represented and surfaced in the Go backend.

---

## Bootstrap status

Landed in code: `internal/apperr` + `handler.WriteError` / `RequestID` / `Recover`.
HTTP mapping and JSON shape match the sections below; adjust codes as product features grow.

---

## Principles

1. **Domain errors are values** in `internal/domain` (or `internal/apperr`), not ad-hoc strings at every call site.
2. **Handlers map domain errors → protocol errors** (HTTP status or WS error frame). Services do not set HTTP status codes.
3. **Wrap with context, inspect with `errors.Is` / `errors.As`.**
4. **Never leak internal driver messages** to clients.

---

## Error categories

| Kind | Example | Client visibility |
|------|---------|-------------------|
| Validation | bad cursor, empty body | yes — field messages |
| AuthN | missing/invalid token | yes — generic unauthorized |
| AuthZ / ACL | not a conversation member | yes — forbidden / not found (avoid existence leaks when required) |
| Not found | message/conversation missing | yes |
| Conflict | duplicate `client_msg_id`, version conflict | yes |
| Rate limit | send too fast | yes — with retry hint |
| Dependency | DB/Redis/MQ down | no internal detail — 503 / retryable frame |
| Programmer | invariant broken | log + 500; no stack to client |

---

## Suggested domain error shape

```go
// bootstrap sketch — place in internal/domain or internal/apperr
var (
    ErrNotFound      = errors.New("not found")
    ErrUnauthorized  = errors.New("unauthorized")
    ErrForbidden     = errors.New("forbidden")
    ErrInvalid       = errors.New("invalid")
    ErrConflict      = errors.New("conflict")
    ErrRateLimited   = errors.New("rate limited")
)

type Error struct {
    Code    string // stable machine code, e.g. "message.not_found"
    Message string // safe for clients
    Err     error  // underlying
}
```

Stable **machine codes** help frontend i18n and WS clients. Prefer `resource.reason` style.

---

## Propagation rules

```text
repo  -- wraps sql.ErrNoRows → domain NotFound
      -- wraps other SQL     → internal error (log cause)

service -- adds business meaning (ACL, conflict)
       -- does not log routine NotFound at error level

handler/ws -- maps to HTTP/WS
           -- logs unexpected errors with request_id / conn_id
```

### HTTP mapping (bootstrap)

| Domain | HTTP |
|--------|------|
| Invalid | 400 |
| Unauthorized | 401 |
| Forbidden | 403 |
| NotFound | 404 |
| Conflict | 409 |
| RateLimited | 429 |
| Dependency / unknown | 503 / 500 |

JSON body sketch:

```json
{
  "error": {
    "code": "conversation.forbidden",
    "message": "not allowed to access this conversation",
    "request_id": "..."
  }
}
```

### WebSocket mapping

Return an error frame correlated by `request_id` when the client sent one. For unsolicited failures (push path), log and close only if the connection is broken; do not drop the TCP connection on routine application errors.

---

## Context and cancellation

- Honor `context.Canceled` / `DeadlineExceeded`: treat as non-500 when the client went away.
- Do not turn client cancel into noisy error alerts.

---

## Panic

- HTTP: recover middleware → log stack → 500.
- WS: recover per connection handler → log → close conn cleanly.
- Never use panic for expected control flow.

---

## Anti-patterns

- `fmt.Errorf("db error: %v", err)` returned straight to JSON clients.
- Catch-all `if err != nil { return err }` at handler without mapping.
- Using HTTP codes inside `service` methods.
- Ignoring idempotent conflict on duplicate message send (should be success or explicit conflict policy, not 500).

---

## Verification

- Table-driven tests: each domain error → expected HTTP status/code.
- WS tests: invalid send returns error frame and keeps connection open.
