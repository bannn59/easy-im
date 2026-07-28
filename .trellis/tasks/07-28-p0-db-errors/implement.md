# Implement: P0 DB + errors

## Checklist

1. [x] Extend `internal/config` with `DatabaseURL`; tests.
2. [x] Add `internal/apperr` (type, sentinels, helpers).
3. [x] Add `internal/db` Open/Ping/Close with pgx.
4. [x] HTTP helpers: RequestID middleware, WriteError, Recover.
5. [x] Wire mux: healthz + readyz; keep CORS.
6. [x] Wire `cmd/api` with optional pool from config.
7. [x] Add goose migrations + `users` init SQL; `cmd/migrate` or make target.
8. [x] Root/backend `docker-compose.yml` for Postgres.
9. [x] Update `backend/README.md` (+ root README one-liner if needed).
10. [x] `go test ./...`; manual migrate up + readyz with compose.
11. [x] Light spec touch: error-handling / database bootstrap status → real paths (optional small).

## Validation

```bash
cd backend && go test ./...
cd backend && go build -o /tmp/easy-im-api ./cmd/api

docker compose up -d
export DATABASE_URL='postgres://easyim:easyim@localhost:5433/easyim?sslmode=disable'
go run ./cmd/migrate up
DATABASE_URL=... PORT=8080 go run ./cmd/api
curl -sf localhost:8080/healthz
curl -sf localhost:8080/readyz
```

## Review gates

- No auth routes.
- apperr does not import net/http.
- Migration reversible or down file present per goose practice.
- Secrets only in env / compose defaults for **local dev** (documented).
