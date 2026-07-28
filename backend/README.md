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

With database + auth:

```bash
# from repo root
docker compose up -d
export DATABASE_URL='postgres://easyim:easyim@localhost:5433/easyim?sslmode=disable'
export AUTH_JWT_SECRET='easyim-dev-secret-change-me'   # required for auth (dev only)
# or: export AUTH_DEV_INSECURE=1   # uses the same default secret
cd backend
go run ./cmd/migrate up
DATABASE_URL="$DATABASE_URL" AUTH_JWT_SECRET="$AUTH_JWT_SECRET" go run ./cmd/api
```

### Auth API (M1)

```bash
# register
curl -s -X POST localhost:8080/v1/auth/register \
  -H 'Content-Type: application/json' \
  -d '{"email":"you@example.com","password":"password12"}'

# login
curl -s -X POST localhost:8080/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"email":"you@example.com","password":"password12"}'

# me
curl -s localhost:8080/v1/me -H "Authorization: Bearer $TOKEN"
```

Env: `AUTH_JWT_SECRET` (or `AUTH_DEV_INSECURE=1`), optional `AUTH_TOKEN_TTL` (Go duration, default `168h`).

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
