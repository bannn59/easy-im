# easy-im

> **[简体中文](README.zh-CN.md)** · English

A self-hostable instant-messaging monorepo. Go backend (stdlib HTTP + PostgreSQL + Kafka) with a React/TypeScript frontend.

| Path | Stack | Status |
|------|--------|--------|
| `backend/` | Go 1.25, stdlib HTTP, PostgreSQL (pgx), Kafka (franz-go), Web Push | Active — chat, groups, realtime, search, push |
| `frontend/` | Vite + React 18 + TypeScript | Active — workspace, chat, search, PWA push |
| `.trellis/` | Trellis workflow + coding specs | Active |

## Features

- **Auth**: register / login / logout with HttpOnly-cookie sessions, profile (display name), password change, zh-CN / en i18n.
- **Conversations**: 1:1 (from Friends) and group chats; conversation list with preview, unread badge, and time.
- **Groups**: create group, add / kick / leave members, transfer owner, rename group — all broadcast live over WebSocket.
- **Messaging**: send (idempotent via `client_msg_id`), paginated history, reply, edit, recall, read receipts, typing indicator, emoji.
- **Realtime**: single app-wide WebSocket. Multi-node fanout over Kafka (`im.messages` / `im.presence`) — message and group events reach members on any API node.
- **Search**: in-conversation search with jump-to-message, plus global cross-conversation search (ACL-scoped) with keyword highlighting.
- **Push**: Web Push offline delivery via the worker; PWA service worker + push settings toggle.
- **Observability**: Prometheus `/metrics`, structured JSON logs, request IDs, unified error envelope.

## Architecture

```
┌──────────┐  HTTP / WS   ┌───────────────────────┐      ┌──────────────┐
│ Frontend │ ───────────► │  cmd/api (per node)    │ ───► │ PostgreSQL   │
│ (React)  │              │  REST + WS hub + auth  │      └──────────────┘
└──────────┘              │  fanout consumer (MQ)  │
                          └──────────┬────────────┘
                                     │ Kafka: im.messages / im.presence
                          ┌──────────▼────────────┐
                          │  cmd/worker            │  offline Web Push
                          └───────────────────────┘
```

- **Transport**: HTTP JSON API for writes/queries; WS (`/v1/ws`) for live push + typing commands. Multi-node delivery via per-node Kafka fanout consumers (origin-skip to avoid double delivery).
- **Primary store**: PostgreSQL (migrations via goose). **Async bus**: Kafka for cross-node fanout and offline push.

## Quick start

Requires Docker (Postgres + Kafka) and Go ≥1.25 + Node ≥18.

### 1. Infrastructure

```bash
docker compose up -d
```

### 2. Migrations

```bash
export DATABASE_URL='postgres://easyim:easyim@localhost:5433/easyim?sslmode=disable'
cd backend && go run ./cmd/migrate up
```

### 3. API

```bash
cd backend
export DATABASE_URL='postgres://easyim:easyim@localhost:5433/easyim?sslmode=disable'
export AUTH_JWT_SECRET='easyim-dev-secret-change-me'   # dev secret; set a real one elsewhere
export KAFKA_BROKERS='localhost:19092'                  # optional; multi-node realtime + push
go run ./cmd/api
# GET http://localhost:8080/healthz  → {"status":"ok"}
# GET http://localhost:8080/readyz   → {"status":"ok"} (DB up)
```

Optional env: `PORT` (default 8080), `CORS_ALLOWED_ORIGINS`, `METRICS_ADDR` (Prometheus), `COOKIE_SECURE`, `COOKIE_DOMAIN`.

### 4. Worker (offline push)

```bash
cd backend
DATABASE_URL="$DATABASE_URL" KAFKA_BROKERS='localhost:19092' \
  VAPID_PUBLIC_KEY=... VAPID_PRIVATE_KEY=... PUSH_SUBJECT=mailto:you@example.com \
  go run ./cmd/worker
```

Push is optional. Without VAPID keys the API still runs; only offline push is disabled.

### 5. Web

```bash
cd frontend
npm install
npm run dev
# http://localhost:5173
```

Optional: copy `frontend/.env.example` to `frontend/.env` and set `VITE_API_BASE` if the API is not on `http://localhost:8080`.

## Test / build

```bash
cd backend && go test ./...
cd frontend && npm run typecheck && npm run build
```

## API surface (selected)

| Method | Path | Purpose |
|--------|------|---------|
| `POST` | `/v1/auth/register` `/v1/auth/login` `/v1/auth/logout` | Auth (cookie session) |
| `GET/PATCH` | `/v1/me`, `/v1/me/profile` | Session, profile |
| `POST` | `/v1/me/password` | Change password |
| `GET` | `/v1/conversations` | Conversation list |
| `POST` | `/v1/conversations/groups` | Create group |
| `GET` | `/v1/conversations/{id}` | Conversation detail |
| `POST` | `/v1/conversations/{id}/messages` | Send message |
| `GET` | `/v1/conversations/{id}/messages` | History / `around_seq` window |
| `GET` | `/v1/conversations/{id}/messages/search` | In-conversation search |
| `PATCH` | `/v1/conversations/{id}` | Rename group (owner) |
| `POST/DELETE` | `/v1/conversations/{id}/members*`, `/owner` | Member management |
| `GET` | `/v1/search/messages` | Global search (ACL-scoped) |
| `GET` | `/v1/friends*` | Friends & requests |
| `GET` | `/v1/ws` | WebSocket (realtime) |
| `GET` | `/metrics` | Prometheus |

## Specs

Coding conventions live under `.trellis/spec/` (backend, frontend, guides). They are source-backed and updated as real patterns land.

## Agents

See `AGENTS.md` for Trellis-oriented assistant instructions.
