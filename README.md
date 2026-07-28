# easy-im

Greenfield instant-messaging monorepo (scaffold stage).

| Path | Stack | Status |
|------|--------|--------|
| `backend/` | Go, stdlib HTTP | `cmd/api` + `GET /healthz` |
| `frontend/` | Vite + React + TypeScript | Shell + home + API health probe |
| `.trellis/` | Trellis workflow + coding specs | Active |

No chat product features yet (auth, conversations, messaging, WebSocket gateway).

## Quick start

### API

```bash
cd backend
go run ./cmd/api
# http://localhost:8080/healthz → {"status":"ok"}
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
