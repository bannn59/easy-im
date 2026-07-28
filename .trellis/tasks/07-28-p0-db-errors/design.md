# Design: P0 Postgres + errors (T1)

## Decisions

| Topic | Choice | Rationale |
|-------|--------|-----------|
| Migration tool | **goose** (SQL migrations) | Single binary via `go run`, simple up/down, fits `backend/migrations/` |
| Driver | **jackc/pgx/v5** pool | Spec preference; use directly or via stdlib if needed later |
| Error package | **`internal/apperr`** | Keep `domain` for entities later; apperr has no HTTP import |
| HTTP mapping | `internal/handler` or `internal/httpx` helper | Map `apperr` → status + JSON |
| Request ID | Header `X-Request-ID`; echo on response; put in error JSON | Align logging guidelines |
| API without DB | **Allow listen** if `DATABASE_URL` empty: healthz works; log warn; **readyz** fails if implemented | Local FE probe unbroken |
| Users table in T1 | **Yes, minimal** | Serial T2 needs migration owner now |
| Compose | `docker-compose.yml` at repo root: Postgres 16, port 5432, volume | Docker available on dev machine |

## Packages

```text
backend/
  cmd/api/main.go           # wire config, optional pool, middleware, mux
  cmd/migrate/main.go       # goose up/down using DATABASE_URL (optional but useful)
  internal/config/          # PORT, DATABASE_URL
  internal/db/              # Open pool, Ping, Close
  internal/apperr/          # Error type, sentinels, Is
  internal/handler/         # healthz, readyz?, request id + recover + encode error
  migrations/
    20260728120000_init.sql # + .down.sql if goose pair style
  docker-compose.yml        # or repo root
```

Goose default: timestamp SQL files in `migrations/`.

## Error JSON

```json
{
  "error": {
    "code": "internal",
    "message": "internal server error",
    "request_id": "…"
  }
}
```

Mapping (from spec):

| apperr kind | code example | HTTP |
|-------------|--------------|------|
| Invalid | `invalid_argument` | 400 |
| Unauthorized | `unauthorized` | 401 |
| Forbidden | `forbidden` | 403 |
| NotFound | `not_found` | 404 |
| Conflict | `conflict` | 409 |
| other/internal | `internal` | 500 |

Handlers use `httpx.WriteError(w, r, err)` so panic recover also goes through same encoder.

## Users migration (minimal for T2)

```sql
CREATE TABLE users (
  id            UUID PRIMARY KEY,
  email         TEXT NOT NULL,
  password_hash TEXT NOT NULL,
  created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX uq_users_email ON users (email);
```

ID generation strategy for T2: app-side UUIDv4/v7 (document in migration comment or README). No trigger required in T1.

## Readyz (optional recommended)

- `GET /healthz` — liveness, no DB  
- `GET /readyz` — `pool.Ping`; 503 if no pool or ping fail  

## Middleware order

```text
RequestID → Recover → (logging optional) → mux
CORS keep as today on outer or after RequestID
```

## Testing

- `apperr` + WriteError table tests (httptest)  
- config DATABASE_URL parse  
- No mandatory docker in unit tests  

## Risks

- Goose vs golang-migrate: pick goose and document; don't dual-support.  
- Empty DATABASE_URL in prod misconfig: readyz catches; document.  
