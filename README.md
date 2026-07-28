# easy-im

Greenfield instant-messaging monorepo (scaffold stage).

| Path | Stack | Status |
|------|--------|--------|
| `backend/` | Go, stdlib HTTP | API + `/healthz` + `/readyz` + auth (M1) |
| `frontend/` | Vite + React + TypeScript | Shell + login/register + `/app` |
| `.trellis/` | Trellis workflow + coding specs | Active |

No chat product features yet (auth, conversations, messaging, WebSocket gateway).

## Quick start

### Postgres (optional for /readyz and future auth)

```bash
docker compose up -d
export DATABASE_URL='postgres://easyim:easyim@localhost:5433/easyim?sslmode=disable'
cd backend && go run ./cmd/migrate up
```

### API

```bash
cd backend
export DATABASE_URL='postgres://easyim:easyim@localhost:5433/easyim?sslmode=disable'
export AUTH_JWT_SECRET='easyim-dev-secret-change-me'
go run ./cmd/api
# http://localhost:8080/healthz → {"status":"ok"}
# with DATABASE_URL: http://localhost:8080/readyz → {"status":"ok"}
# auth: POST /v1/auth/register|login, GET /v1/me
```

### Web

```bash
cd frontend
npm install
npm run dev
# http://localhost:5173
```

Optional: copy `frontend/.env.example` to `frontend/.env` and set `VITE_API_BASE`.

## Specs

Coding conventions live under `.trellis/spec/` (backend, frontend, guides). They started as bootstrap assumptions; update them as real patterns land.

## Agents

See `AGENTS.md` for Trellis-oriented assistant instructions.
