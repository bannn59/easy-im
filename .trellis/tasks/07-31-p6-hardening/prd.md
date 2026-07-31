# P6 Production CORS & Auth Hardening

## Goal

Eliminate the known production-security debt: hardcoded CORS `*`, JWT in localStorage, and JWT in WS query string. Migrate to HttpOnly cookie sessions with tightened CORS and WS origin validation.

## Background

The scaffold shipped with documented dev-only defaults that are unsafe for production: `Access-Control-Allow-Origin: *`, JWT in `localStorage` (XSS-exposed), WS auth via `?token=` (leaks into logs/history), and `CheckOrigin: true` on the WS upgrader. This task closes those gaps.

### Current technical foundation (confirmed by research)

- **CORS**: `withCORS` in `handler/router.go` sets `*`, hand-rolled, no allowlist/env, no `Vary: Origin`
- **JWT**: HS256, claims `sub`+`email`+`iat`+`exp`, TTL default 168h; issued in `AuthService.issueToken`
- **Storage**: frontend `localStorage` (`TOKEN_KEY = 'easyim_access_token'`), sent as `Authorization: Bearer`
- **WS auth**: token in query string, server falls back to bearer; `CheckOrigin` returns `true`
- **Env**: `AUTH_JWT_SECRET` (empty → 503), `AUTH_DEV_INSECURE=1` hardcoded dev secret, `AUTH_TOKEN_TTL`, `PORT`
- **No CSRF, no cookie session, no refresh token, no Origin validation**

## Scope (confirmed)

- **Cookie 会话迁移**（方案 B）：HttpOnly cookie 存 JWT（方案 A：cookie 直接存 JWT）
- **CSRF 防护**：仅 `SameSite=Lax`（方案 A）
- **CORS 收紧**：env 驱动 allowlist + `Vary: Origin` + `Access-Control-Allow-Credentials`
- **WS 认证重做**：从 cookie 读 JWT + `CheckOrigin` 校验 Origin
- **保留**：多设备语义、无状态 token、现有 JWT 校验逻辑

## Requirements

### R1 — Cookie 会话

- 登录/注册成功后，后端通过 `Set-Cookie` 下发 `easyim_session`（HttpOnly、SameSite=Lax、Secure（生产）、Path=/）
- Cookie 值 = 现有 JWT access token
- `require` 中间件和 WS handler **只从 cookie 读 token**（不保留 Authorization header 兼容）
- 登出：`POST /v1/auth/logout` 清除 cookie
- **Token TTL 默认缩短为 24h**

### R2 — CSRF 防护

- `SameSite=Lax` 阻断跨站请求带 cookie
- 不做 CSRF token（方案 A）

### R3 — CORS 收紧

- `Access-Control-Allow-Origin` 来自 env `CORS_ALLOWED_ORIGINS`（逗号分隔白名单）
- 动态回显请求 Origin（若在名单内），否则不设 CORS 头
- `Vary: Origin`（响应随 Origin 变化）
- `Access-Control-Allow-Credentials: true`（cookie 方案必须）
- 未匹配 Origin 的跨站请求拒绝（不设 CORS 头即可让浏览器阻止）

### R4 — WS 认证

- WS handler 从 cookie 读 JWT（`r.Cookie`），不再用 query string
- `CheckOrigin`：校验请求 Origin 在 CORS 白名单内
- 前端 WS 连接不再拼 `?token=`（同源 cookie 自动携带）

### R5 — 密钥加固

- `AUTH_DEV_INSECURE=1` 硬编码 secret：保留但文档化（仅 dev），生产必须显式设 secret
- 启动时若 `AUTH_DEV_INSECURE` 未设且 secret 为空 → 拒绝启动（而非静默 503）
- **TTL 默认 24h**（从 168h 缩短）

## Acceptance Criteria

- [ ] 登录后浏览器有 `easyim_session` cookie（HttpOnly、SameSite=Lax、生产 Secure）
- [ ] cookie 中的 JWT 可正常认证所有受保护路由（替代 Authorization header）
- [ ] 登出后 cookie 清除，受保护路由 401
- [ ] 跨站请求（不同 Origin）不携带 cookie / 被浏览器阻止（CSRF 防护生效）
- [ ] CORS 白名单外 Origin 的请求无 CORS 头（浏览器拦截响应）
- [ ] WS 连接同源可认证成功，跨源被拒绝（CheckOrigin）
- [ ] 前端不再把 JWT 存 localStorage（Session 改造后无 token 残留）
- [ ] 前端 WS URL 不含 `?token=`
- [ ] 生产（无 AUTH_DEV_INSECURE）空 secret 时拒绝启动或清晰报错
- [ ] 现有功能回归：登录/注册/消息/会话/好友/设置/实时

## Out of Scope

- Opaque session 存储 / 服务端吊销（保留无状态 JWT）
- Refresh token 轮换
- CSRF token（SameSite=Lax 已覆盖单域场景）
- 多域部署（子域 cookie 共享）
- 速率限制（spec 建议，但独立项）
- 部署清单（Docker/nginx 配置，另一任务）

## Open Questions

_(none — all blocking decisions resolved)_
