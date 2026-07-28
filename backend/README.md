# easy-im backend

Go module for the easy-im API (and later gateway/worker processes).

## Run API

```bash
cd backend
go run ./cmd/api
# GET http://localhost:8080/healthz
```

Optional: `PORT=8081 go run ./cmd/api`

## Test / build

```bash
go test ./...
go build -o bin/api ./cmd/api
```

## Layout

See `.trellis/spec/backend/directory-structure.md`.

- `cmd/api` — HTTP API (this scaffold)
- `cmd/gateway` / `cmd/worker` — not scaffolded yet (WebSocket / jobs)
- `internal/handler` — HTTP handlers
- `internal/config` — env config
- `migrations/` — SQL migrations (empty)

No IM business features in this scaffold — health check only.
