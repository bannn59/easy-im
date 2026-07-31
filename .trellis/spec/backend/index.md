# Backend Development Guidelines

> Coding guidance for the Go backend of **easy-im**.

---

## Bootstrap status

> Specs mix **source-backed** rules (auth, conversations, messages, in-process hub WS, `reply_to`)
> with still-planned multi-node gateway/MQ assumptions. Prefer code paths under `backend/internal`
> when a bootstrap paragraph disagrees.

| Assumption | Choice |
|------------|--------|
| Language | Go (modules) |
| Repo shape | Monorepo (`backend/` + `frontend/`) |
| Transport | HTTP JSON API + WS push (`/v1/ws` on api process for now) |
| Primary store | PostgreSQL (pgx + goose) |
| Cache / presence | Redis (planned; not required for text MVP) |
| Async fan-out | In-process hub now; NATS/Kafka later |

---

## Guidelines index

| Guide | Description | Status |
|-------|-------------|--------|
| [Directory Structure](./directory-structure.md) | `cmd/`, `internal/`, package boundaries | Bootstrap |
| [Database Guidelines](./database-guidelines.md) | SQL, migrations, messages + reply_to | Source-backed |
| [Auth & Session](./auth-session.md) | Cookie sessions, JWT, CORS | Source-backed |
| [Realtime & Messaging](./realtime-messaging.md) | Hub WS, message DTO, reply_to scenario | Source-backed |
| [Error Handling](./error-handling.md) | Domain errors → HTTP | Source-backed |
| [Logging Guidelines](./logging-guidelines.md) | Structured logs, request/conn IDs | Bootstrap |
| [Quality Guidelines](./quality-guidelines.md) | Forbidden patterns, tests, review | Bootstrap |

---

## How to use these guidelines

1. Read **Directory Structure** before creating packages or entrypoints.
2. For features that touch storage, also read **Database** and **Error Handling**.
3. For chat delivery or new message fields, read **Realtime & Messaging** (HTTP/WS parity).
4. Prefer updating these files when implementation diverges — do not leave dead rules.

**Language**: Spec documents are written in **English**. Product UI copy may be localized separately.
