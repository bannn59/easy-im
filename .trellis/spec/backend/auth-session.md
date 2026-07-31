# Auth & Session

> Authentication and session conventions for easy-im backend.

---

## Bootstrap status

**Landed**: cookie-based sessions. Login/register set an HttpOnly session cookie holding
the access-token JWT; all protected routes authenticate from that cookie. No
Authorization-header auth, no localStorage token, no refresh token.

---

## Session cookie

| Attribute | Value | Note |
|-----------|-------|------|
| Name | `easyim_session` | |
| Value | JWT access token | Same token as pre-cookie auth |
| Path | `/` | |
| HttpOnly | `true` | JS cannot read it (XSS-safe) |
| SameSite | `Lax` | Blocks cross-site cookie attach (CSRF base) |
| Secure | `COOKIE_SECURE=1` | HTTPS-only in production |
| Domain | `COOKIE_DOMAIN` | Optional parent-domain scope |
| Max-Age | — | Cookie dies with the JWT `exp` |

- **Only auth source**: `RequireUser`, `/v1/me`, and the WS upgrader read the token
  from this cookie. No Bearer header fallback.
- **No CSRF token**: `SameSite=Lax` covers the single-origin SPA. Add double-submit
  CSRF only if multi-origin/subdomain sharing is introduced.
- **Logout**: `POST /v1/auth/logout` clears the cookie (frontend JS cannot delete
  HttpOnly cookies).

## Endpoints

| Method | Path | Auth | Purpose |
|--------|------|------|---------|
| POST | `/v1/auth/register` | — | Create account; sets session cookie |
| POST | `/v1/auth/login` | — | Login; sets session cookie |
| POST | `/v1/auth/logout` | — | Clear session cookie |
| GET | `/v1/me` | cookie | Profile (id, email, display_name, created_at) |
| PATCH | `/v1/me/profile` | cookie | Update display name |
| POST | `/v1/me/password` | cookie | Change password (current password required) |

## JWT

- HS256; claims `sub` (user id), `email`, `iat`, `exp`.
- **TTL default 24h** (`AUTH_TOKEN_TTL` overridable). No refresh token — a new login
  is the only renewal path.
- Multi-device: each device keeps its own cookie; no server-side revocation
  (opaque sessions out of scope).

## CORS

- `CORS_ALLOWED_ORIGINS` (comma-separated) is the only Origin allowlist; default
  `http://localhost:5173` (dev frontend).
- Middleware echoes the allowed Origin, sets `Access-Control-Allow-Credentials: true`,
  and always `Vary: Origin`. Unlisted Origins get no CORS headers (browser blocks).
- WS `CheckOrigin` uses the same allowlist.

## Key material

- `AUTH_JWT_SECRET` is **required** to start; empty secret aborts startup unless
  `AUTH_DEV_INSECURE=1` (dev-only hardcoded default).
- Secrets are never logged.
