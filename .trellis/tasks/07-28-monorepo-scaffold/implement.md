# Implement: monorepo scaffold

## Checklist

### Backend

1. [x] Create `backend/go.mod` (`module easy-im/backend`).
2. [x] `internal/config` — load `PORT`.
3. [x] `internal/handler` — mux + `Healthz`.
4. [x] `cmd/api/main.go` — listen, shutdown.
5. [x] `backend/README.md` — run instructions.
6. [x] `migrations/.gitkeep`.
7. [x] `go test ./...` and `go build ./cmd/api`.
8. [x] Manual or scripted curl `/healthz`.

### Frontend

9. [x] Scaffold Vite React-TS under `frontend/` (`npm create vite@latest` or equivalent files).
10. [x] Reshape `src/` to `app/`, `api/`, `realtime/`, `features/`, `shared/`, `styles/`.
11. [x] Routes `/` (+ optional `/health` probe).
12. [x] `VITE_API_BASE` in `.env.example`.
13. [x] `frontend/README.md`.
14. [x] `npm install && npm run build`.

### Repo

15. [x] Root `README.md` with both run commands.
16. [x] Ensure `.gitignore` covers `node_modules`, build outputs (already partly present).
17. [x] Optional: one-line note in spec bootstrap status that dirs now exist (only if accurate).

## Validation commands

```bash
cd backend && go test ./...
cd backend && go build -o /tmp/easy-im-api ./cmd/api
# terminal A
PORT=8080 /tmp/easy-im-api &
curl -sf http://127.0.0.1:8080/healthz
kill %1 2>/dev/null || true

cd frontend && npm install && npm run build
```

## Review gates

- No IM business handlers pretending to be done.
- No secrets.
- Handler does not import non-existent repo packages.

## Rollback

Delete `backend/` and `frontend/` if abandoned; task research elsewhere unaffected.
