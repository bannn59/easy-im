# Design: T2 Auth (register / login / me)

## Decisions

| Topic | Choice | Rationale |
|-------|--------|-----------|
| Password | **bcrypt** (`golang.org/x/crypto/bcrypt`, cost 10) | Simple, good enough for MVP |
| Token | **JWT HS256** | Stateless; fits single API node; secret from env |
| Token TTL | `AUTH_TOKEN_TTL` default **168h** (7d) dev | Configurable duration |
| User id | **UUIDv4** generated in service | Matches `users.id UUID` |
| Email | Store **lowercased** trimmed | Unique index friendly |
| Login errors | Always **401 unauthorized** | Reduce account enumeration |
| API prefix | `/v1/auth/*`, `/v1/me` | Room for versioning |
| FE token storage | `localStorage` key `easyim_access_token` | MVP; document XSS caveat |
| Protected UI | `/app` shell after login; `/` marketing/home can stay public or redirect — **Home public, `/app` protected** |

## Config additions

```text
DATABASE_URL          # required for auth routes (pool already)
AUTH_JWT_SECRET       # required in practice; dev default only if AUTH_DEV_INSECURE=1
AUTH_TOKEN_TTL        # optional, Go duration string, default 168h
```

Startup: if `AUTH_JWT_SECRET` empty and not insecure flag → log error and **refuse to start** auth-capable API **or** allow start but auth returns 503. Prefer: **require secret** when building production mindset; for local DX set compose/README default `easyim-dev-secret-change-me` + warning.

## Packages

```text
internal/domain/user.go          # User struct (id, email, timestamps) — no hash in public DTO
internal/repo/user_repo.go       # Create, FindByEmail, FindByID
internal/service/auth_service.go # Register, Login, ParseToken, Me
internal/handler/auth.go         # HTTP bind
internal/handler/auth_middleware.go # optional Bearer extract for /me
internal/config                  # JWT secret, TTL
```

## HTTP contracts

### POST /v1/auth/register

Request: `{ "email": "...", "password": "..." }`  
Response 201: `{ "access_token": "...", "token_type": "Bearer", "user": { "id", "email" } }`  
Errors: 400 invalid, 409 conflict, 503 no db

### POST /v1/auth/login

Same body; 200 + token; 401 on failure; 503 no db

### GET /v1/me

Header: `Authorization: Bearer <jwt>`  
200: `{ "id", "email" }`  
401 missing/invalid

## JWT claims

```text
sub: user id (UUID string)
email: optional convenience
iat, exp: standard
```

## Frontend

```text
src/features/auth/   # LoginPage, RegisterPage or AuthPage
src/api/auth.ts      # register, login, me
src/api/http.ts      # fetch wrapper with token + error parse
src/app/Session.tsx  # context provider
src/app/AppShell.tsx # protected layout
```

CORS: allow `Authorization`.

## Testing

- Auth service with fake repo interface **or** integration with real PG (compose) — prefer **interface + fake** for unit speed + one optional integration.
- Handler tests with httptest + stub service.
- FE: typecheck/build; manual curl script in README.

## Security notes (MVP)

- HTTPS assumed in prod later.
- localStorage XSS → full account; acceptable for scaffold IM, track as debt.
- No refresh rotation.
