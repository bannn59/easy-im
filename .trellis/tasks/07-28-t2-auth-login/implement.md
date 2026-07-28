# Implement: T2 Auth

## Checklist

### Backend

1. [x] Config: `AUTH_JWT_SECRET`, `AUTH_TOKEN_TTL`; tests.
2. [x] `domain.User` public fields; never expose hash.
3. [x] `repo.UserRepo` Create / ByEmail / ByID (pgx).
4. [x] `service.AuthService` Register / Login / Me / token issue+parse.
5. [x] Handlers + route registration on mux; CORS Authorization.
6. [x] Require pool for auth routes (503).
7. [x] Unit tests (fake repo or service); `go test ./...`.
8. [x] README curl examples + env.

### Frontend

9. [x] `api/http.ts` + `api/auth.ts` with error code parsing.
10. [x] Session context (token in localStorage).
11. [x] Login + Register pages; `/app` protected shell.
12. [x] Nav updates; logout.
13. [x] `npm run build`.

### Docs / task

14. [x] Root or backend README auth section.
15. [x] Tick PRD AC.

## Validation

```bash
docker compose up -d
export DATABASE_URL='postgres://easyim:easyim@localhost:5433/easyim?sslmode=disable'
export AUTH_JWT_SECRET='easyim-dev-secret-change-me'
cd backend && go run ./cmd/migrate up && go test ./...
DATABASE_URL=... AUTH_JWT_SECRET=... go run ./cmd/api
cd frontend && npm run build
```
