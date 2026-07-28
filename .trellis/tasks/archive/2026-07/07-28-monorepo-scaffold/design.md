# Design: monorepo scaffold

## Goals in one sentence

Ship two runnable roots (`backend/`, `frontend/`) that match bootstrap directory rules, with zero IM business logic.

## Backend

### Module

- Path: `easy-im/backend` (no public remote yet; rename later if GitHub path is fixed).
- Go version: match toolchain (`go 1.22`+ in go.mod; local is 1.26).

### Layout (MVP of spec)

```text
backend/
├── go.mod
├── cmd/api/main.go          # signals, listen, wire
├── internal/
│   ├── config/config.go     # PORT, optional APP_ENV
│   ├── handler/health.go    # GET /healthz
│   ├── handler/router.go    # stdlib mux or ServeMux
│   ├── app/api.go           # optional NewAPIServer() wire
│   └── domain/              # optional doc.go only
├── migrations/.gitkeep      # empty placeholder
└── README.md
```

**Not in this task:** `cmd/gateway`, `cmd/worker` binaries (mention in README as future). Avoid empty packages that break `go build ./...` — use `.gitkeep` only outside Go packages, or `doc.go` for intentional packages.

### HTTP

- Stdlib `net/http` only (no Gin/Echo yet) to keep deps zero.
- `GET /healthz` → `200` `application/json` `{"status":"ok"}`.
- Graceful shutdown on SIGINT/SIGTERM.

### Config

- `PORT` env, default `8080`.
- No config files required for MVP.

## Frontend

### Stack

- Vite + `react` + `react-dom` + TypeScript.
- `react-router-dom` for `src/app` routes.
- npm (already available).

### Layout (MVP of spec)

```text
frontend/
├── package.json
├── vite.config.ts
├── tsconfig.json / tsconfig.app.json
├── index.html
├── src/
│   ├── main.tsx
│   ├── app/App.tsx          # router + layout shell
│   ├── app/routes.tsx       # route table
│   ├── features/            # .gitkeep or README only
│   ├── shared/ui/           # optional Placeholder page chrome
│   ├── api/client.ts        # baseURL from import.meta.env
│   ├── realtime/index.ts    # export placeholder; no WebSocket yet
│   └── styles/index.css     # minimal
└── README.md
```

### Routes (user-perceptible but non-IM)

| Path | Page | Purpose |
|------|------|---------|
| `/` | Home | 说明 easy-im scaffold + 链到 health 说明 |
| `/health` | HealthProbe (optional) | 浏览器侧 fetch `${VITE_API_BASE}/healthz` 展示结果（失败也友好） |

Env: `VITE_API_BASE` default `http://localhost:8080`.

### Styling

- One plain CSS file; no Tailwind decision in this task (spec left styling TBD).

## Root docs

- Short `README.md` at repo root: what this monorepo is, how to run API + web.
- Keep `AGENTS.md` Trellis block untouched.

## Testing / verification

| Check | Command |
|-------|---------|
| Go build | `cd backend && go build -o /tmp/easy-im-api ./cmd/api` |
| Go test | `cd backend && go test ./...` |
| Health | run binary + `curl -sf localhost:8080/healthz` |
| FE build | `cd frontend && npm ci\|npm install && npm run build` |

## Risks / tradeoffs

| Choice | Tradeoff |
|--------|----------|
| Stdlib HTTP only | Add router later; zero dep friction now |
| No TanStack Query yet | Add when first list feature lands |
| No gateway cmd | Spec wants split early; README documents deferral to avoid fake empty mains |
| Module path `easy-im/backend` | May rename when remote is known |

## Out of design

Docker, CI workflows, OpenAPI generation, real auth.
