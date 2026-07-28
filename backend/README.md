# easy-im backend

Go module for the easy-im API (and later gateway/worker processes).

## Run API

```bash
cd backend
go run ./cmd/api
# GET http://localhost:8080/healthz   (liveness, no DB required)
# GET http://localhost:8080/readyz    (503 if DATABASE_URL unset or DB down)
```

Optional: `PORT=8081 go run ./cmd/api`

With database:

```bash
# from repo root
docker compose up -d
export DATABASE_URL='postgres://easyim:easyim@localhost:5433/easyim?sslmode=disable'
cd backend
go run ./cmd/migrate up
DATABASE_URL="$DATABASE_URL" go run ./cmd/api
```

## Migrations (goose)

SQL files live in `migrations/`. Commands:

```bash
cd backend
export DATABASE_URL='postgres://easyim:easyim@localhost:5433/easyim?sslmode=disable'
go run ./cmd/migrate up
go run ./cmd/migrate status
go run ./cmd/migrate down   # one step
```

Default local credentials are **dev-only** (see repo `docker-compose.yml`).
Host port **5433** maps to container 5432 (avoids clashing with a local Postgres on 5432).

## Errors

API errors use:

```json
{"error":{"code":"not_found","message":"…","request_id":"…"}}
```

`X-Request-ID` is accepted or generated and returned on every response.

## Test / build

```bash
go test ./...
go build -o bin/api ./cmd/api
go build -o bin/migrate ./cmd/migrate
```

## Layout

See `.trellis/spec/backend/directory-structure.md`.

- `cmd/api` — HTTP API
- `cmd/migrate` — goose migrations
- `internal/apperr` — stable error codes
- `internal/db` — pgx pool
- `internal/handler` — HTTP + middleware
- `internal/config` — env config
- `migrations/` — versioned SQL

No auth/login handlers yet (T2). `users` table is created by the init migration for T2.
