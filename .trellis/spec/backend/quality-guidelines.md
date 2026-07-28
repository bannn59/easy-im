# Quality Guidelines

> Code quality bar for the Go backend.

---

## Bootstrap status

Enforce these as the monorepo backend grows. Wire concrete linters in CI when `backend/` exists.

---

## Required patterns

1. **`context.Context` as first parameter** on I/O and service methods.
2. **Interfaces defined near consumers** when mocking is needed; avoid package-wide interface pollution.
3. **Table-driven tests** for codecs, error mapping, pure domain logic.
4. **Explicit migrations** for every schema change.
5. **Idempotent message send** via `client_msg_id`.
6. **ACL on every conversation-scoped operation.**

---

## Forbidden patterns

| Pattern | Why |
|---------|-----|
| Business logic in `handler` or raw WS loop | Untestable, leaks protocol into domain |
| Global mutable conn maps without locking/documentation | Races under load |
| Ignoring `err` from `Close`, encode, or publish | Silent loss |
| `init()` with environment side effects | Hard to test; prefer explicit `cmd` setup |
| Cross-import `internal/handler` → `internal/repo` | Skips use-case layer |
| Shipping with default secrets in repo | Security |
| unbounded goroutines per message without backpressure | Gateway collapse |
| `SELECT *` + map[string]any through layers | Breaks contracts |

---

## Testing requirements

| Layer | Expectation |
|-------|-------------|
| `domain` | Unit tests for validation and pure helpers |
| `service` | Unit tests with fake repo/cache/mq |
| `repo` | Integration tests with testcontainers or CI Postgres |
| `ws` codec | Round-trip + unknown type / version handling |
| `gateway` | Conn registry, disconnect cleanup, dual-node fan-out if feasible |
| `handler` | HTTP status mapping tests |

Name integration tests clearly (`TestIntegration_...`) and gate with build tags or env if they need Docker.

---

## Tooling (bootstrap)

When the module exists, prefer:

```bash
cd backend
go test ./...
go vet ./...
golangci-lint run   # once config is added
```

Suggested linter ideas: `errcheck`, `staticcheck`, `govet`, `ineffassign`, `misspell`.

---

## Code review checklist

- [ ] Package boundary respected (handler/ws → service → repo/cache/mq)?
- [ ] New DB fields have migrations and repo mapping?
- [ ] Errors mapped without leaking internals?
- [ ] Message paths idempotent and ACL-checked?
- [ ] Logs have correlation ids and no secrets/bodies at info?
- [ ] WS/HTTP contract versioning considered?
- [ ] Tests cover happy path + not-found + forbidden + duplicate client id?
- [ ] Specs updated if a new convention was introduced?

---

## Performance notes (IM)

- Measure before adding caches.
- Batch presence and push where possible.
- Avoid per-message JSON reflection-heavy paths in hot encode loops — prefer typed encoders.
- Cap read/write deadlines on every connection.

---

## Anti-patterns in reviews

- “Temporary” direct SQL in handlers that never moves.
- Copy-pasting fan-out logic into both API and gateway.
- Huge PR mixing schema, gateway, and UI contracts without a shared event/DTO owner.
