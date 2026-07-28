# Backend Development Guidelines

> Coding guidance for the Go backend of **easy-im**.

---

## Bootstrap status

> **These specs are bootstrap assumptions, not evidence from existing product code.**
>
> As of bootstrap, this repository has no application source yet. Treat the rules
> below as the intended first-pass conventions for a monorepo IM system. When real
> code lands, re-run `trellis-spec-bootstrap` (or update these files) so every rule
> points at actual paths, packages, and tests.

| Assumption | Choice |
|------------|--------|
| Language | Go (modules) |
| Repo shape | Monorepo (`backend/` + `frontend/` + optional `packages/`) |
| Transport | HTTP JSON API + WebSocket long-lived connections |
| Primary store | PostgreSQL (MySQL acceptable if already standardized) |
| Cache / presence | Redis |
| Async fan-out | Message bus (prefer **NATS** for early stages; Kafka when durability/partition scale is required) |

---

## Guidelines index

| Guide | Description | Status |
|-------|-------------|--------|
| [Directory Structure](./directory-structure.md) | `cmd/`, `internal/`, package boundaries | Bootstrap |
| [Database Guidelines](./database-guidelines.md) | SQL access, migrations, transactions | Bootstrap |
| [Realtime & Messaging](./realtime-messaging.md) | WebSocket gateway, presence, MQ | Bootstrap |
| [Error Handling](./error-handling.md) | Domain errors → HTTP / WS frames | Bootstrap |
| [Logging Guidelines](./logging-guidelines.md) | Structured logs, request/conn IDs | Bootstrap |
| [Quality Guidelines](./quality-guidelines.md) | Forbidden patterns, tests, review | Bootstrap |

---

## How to use these guidelines

1. Read **Directory Structure** before creating packages or entrypoints.
2. For features that touch storage, also read **Database** and **Error Handling**.
3. For chat delivery, presence, or fan-out, read **Realtime & Messaging**.
4. Prefer updating these files when implementation diverges — do not leave dead rules.

**Language**: Spec documents are written in **English**. Product UI copy may be localized separately.
