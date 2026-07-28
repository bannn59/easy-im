# Directory Structure

> How Go backend code is organized in easy-im.

---

## Bootstrap status

Scaffold landed: `backend/go.mod`, `cmd/api`, `internal/{config,handler,app,domain}`, `migrations/`.  
Gateway/worker and full domain packages remain **targets**, not implemented.

---

## Intended monorepo roots

```text
/
├── backend/                 # Go module root (go.mod lives here)
│   ├── cmd/
│   │   ├── api/             # HTTP API process
│   │   ├── gateway/         # WebSocket / long-connection process
│   │   └── worker/          # MQ consumers, offline push, maintenance jobs
│   ├── internal/
│   │   ├── domain/          # pure types + domain errors (no I/O)
│   │   ├── service/         # use-cases / application services
│   │   ├── repo/            # PostgreSQL (or MySQL) persistence
│   │   ├── cache/           # Redis adapters (presence, session, rate limit)
│   │   ├── mq/              # producers / consumers for NATS|Kafka
│   │   ├── gateway/         # connection hub, route tables, heartbeats
│   │   ├── handler/         # HTTP handlers + request DTOs
│   │   ├── ws/              # WS frame codec, auth handshake helpers
│   │   ├── auth/            # token validation, identity context
│   │   ├── config/          # env/file config loading
│   │   └── app/             # wire dependencies per process
│   ├── migrations/          # SQL migrations (golang-migrate or equivalent)
│   ├── api/                 # OpenAPI / proto sources if checked in
│   └── go.mod
├── frontend/                # React + TypeScript app
└── packages/                # optional shared contracts (OpenAPI, proto, JSON schemas)
```

Split `cmd/api` and `cmd/gateway` early even if they share packages. IM load profiles differ: API is request/response; gateway is long-lived connections and fan-out.

---

## Package responsibilities

| Package | Owns | Must not own |
|---------|------|--------------|
| `internal/domain` | Entities, IDs, domain errors, pure validation | SQL, Redis, HTTP, MQ |
| `internal/service` | Use-cases (send message, create conversation, mark read) | Raw SQL, HTTP status codes |
| `internal/repo` | Queries, row mapping, transactions | Business policy beyond persistence |
| `internal/handler` | HTTP binding, status mapping, pagination params | Direct SQL |
| `internal/gateway` + `internal/ws` | Conn lifecycle, frame encode/decode, local routing | Durable message history writes (delegate to service/repo) |
| `internal/mq` | Topic names, serialize/deserialize bus events | UI-facing DTO shaping |
| `cmd/*` | `main`, signal handling, process wiring | Business logic |

---

## Dependency direction

```text
cmd → internal/app → handler / gateway / worker
                         ↓
                      service
                    ↙   ↓   ↘
                 repo  cache  mq
                         ↓
                      domain
```

- `domain` has no outward project imports.
- `handler` and `ws` call `service`, never `repo` directly (except read-only health/admin if explicitly documented).
- Shared code that both API and gateway need lives under `internal/` packages, not under a single `cmd`.

---

## Naming conventions

| Kind | Convention | Example |
|------|------------|---------|
| Packages | short, lowercase, no underscores | `repo`, `gateway` |
| Files | snake or lower descriptive | `message_repo.go`, `conn_hub.go` |
| Exported types | PascalCase noun | `Message`, `ConversationID` |
| Interfaces at consumer | small, use-case shaped | `MessageStore`, `PresenceStore` |
| IDs | typed where practical | `type UserID string` |
| Tests | `*_test.go` co-located; integration under `backend/test/` if heavy | |

Prefer **accept interfaces, return structs** at service boundaries.

---

## Adding a new feature (checklist)

1. Domain types / errors in `internal/domain` if new concepts appear.
2. Persistence in `internal/repo` + migration under `migrations/`.
3. Use-case methods in `internal/service`.
4. HTTP routes in `internal/handler` **and/or** WS frames in `internal/ws` as needed.
5. Async side effects (push, fan-out, search index) via `internal/mq`, not inline in the request path beyond “enqueue”.
6. Wire in `internal/app` and the relevant `cmd/*`.

---

## Anti-patterns

- Putting business rules in HTTP handlers or WS read loops.
- One giant `internal/pkg` dumping ground.
- Importing `frontend/` or generating TS types by scraping Go structs ad hoc — use `packages/contracts` or OpenAPI/proto.
- Starting with many microservices before process boundaries are proven. Prefer multiple `cmd` binaries sharing `internal/` first.

---

## Verification

When code exists:

```bash
cd backend && go test ./...
cd backend && go vet ./...
```

Confirm no `handler` → `repo` imports except documented exceptions:

```bash
# example audit once packages exist
go list -f '{{.ImportPath}} {{.Imports}}' ./internal/handler/...
```
