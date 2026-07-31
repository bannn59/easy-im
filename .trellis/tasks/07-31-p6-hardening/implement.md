# Implement: P6 Production CORS & Auth Hardening

## Implementation Order

后端（config → cookie 常量 → auth handler → require 中间件 → WS → CORS），前端（http credentials → api 层 → Session → 组件）。

---

### Step 1: Config — 新增 env

**Files:** `backend/internal/config/config.go`

- [x] `CORSAllowedOrigins []string`（`CORS_ALLOWED_ORIGINS`，逗号分隔，默认含 `http://localhost:5173`）
- [x] `CookieSecure bool`（`COOKIE_SECURE`，默认 false）
- [x] `CookieDomain string`（`COOKIE_DOMAIN`，可选）
- [x] 启动校验：`AUTH_DEV_INSECURE` 未设 且 `AUTH_JWT_SECRET` 空 → 拒绝启动
- [x] TTL 默认从 168h → 24h

**Verify:** `go build ./...` + config 测试更新

### Step 2: Cookie 常量 + AuthHandler cookie 签发

**Files:** `backend/internal/handler/auth.go`(新 `cookies.go` 或内置)

- [x] `sessionCookieName = "easyim_session"` + `CookieConfig{Secure, Domain}`
- [x] `Login`/`Register` 成功后 `Set-Cookie`（HttpOnly、SameSite=Lax、Secure 可选）
- [x] 响应体去掉 `access_token`
- [x] 新增 `Logout` handler（清除 cookie）
- [x] AuthHandler 构造传 `CookieConfig`

**Verify:** `go build ./...`

### Step 3: require 中间件改 cookie

**Files:** `backend/internal/handler/auth_middleware.go`

- [x] `RequireUser` 从 `r.Cookie(sessionCookieName)` 读 token
- [x] 移除 Authorization header 路径

**Verify:** `go build ./...`

### Step 4: WS 认证 + CheckOrigin

**Files:** `backend/internal/handler/ws.go`

- [x] WSHandler 从 cookie 读 JWT（替换 query/header）
- [x] `CheckOrigin` 校验 Origin ∈ CORS 白名单（空 Origin 允许）
- [x] WSHandler 构造传 allowedOrigins

**Verify:** `go build ./...`

### Step 5: CORS 中间件重写

**Files:** `backend/internal/handler/router.go`

- [x] `withCORS(allowed)` 动态回显 + credentials + `Vary: Origin`
- [x] 移除 `*` 硬编码
- [x] 移除 `Authorization` from allowed headers
- [x] `NewMux` 传入 allowedOrigins
- [x] `app/api.go` 传 config

**Verify:** `go build ./...`

### Step 6: 后端测试

**Files:** `backend/internal/handler/*_test.go`, `backend/internal/config/config_test.go`

- [x] config：CORS_ALLOWED_ORIGINS 解析、CookieSecure 解析、启动校验
- [x] handler：cookie 认证替代 header（登录 → cookie → require 通过）
- [x] CORS：白名单内回显、白名单外无头、credentials

**Verify:** `go test ./...` 全绿

### Step 7: 前端 — http credentials

**Files:** `frontend/src/api/http.ts`

- [x] `fetch` 加 `credentials: 'include'`
- [x] `RequestOptions` 移除/忽略 token（保留参数兼容或移除）

**Verify:** `tsc --noEmit`

### Step 8: 前端 — api 层 token 参数清理

**Files:** `frontend/src/api/*.ts`（auth, conversations, messages, friends, settings）

- [x] 所有 API 函数移除 `token` 参数（或改为忽略）
- [x] 各调用方更新

**Verify:** `tsc --noEmit`

### Step 9: 前端 — Session 重构

**Files:** `frontend/src/app/Session.tsx`

- [x] 移除 `TOKEN_KEY`/localStorage
- [x] `SessionState` 移除 `token`，只保留 `user`
- [x] `login`/`register`：调 API → `fetchMe` 拿 user
- [x] `logout`：调 `POST /v1/auth/logout` → 清 user
- [x] `api/auth.ts` 加 `logout` 函数

**Verify:** `tsc --noEmit`

### Step 10: 前端 — 组件 + WS 更新

**Files:** `frontend/src/app/*.tsx`, `frontend/src/features/*/*.tsx`, `frontend/src/realtime/index.ts`

- [x] 所有 `session.token` 使用点移除（AppShell、ConversationRoom、FriendsPage、SettingsPage）
- [x] `realtime/index.ts`：`connectRealtime` 不再拼 `?token=`；`RealtimeProvider` 用 `user` 判断
- [x] `sendFrame` 不变（WS 已带 cookie）

**Verify:** `tsc --noEmit`

### Step 11: 回归验证

- [x] `go test ./...` 全绿
- [x] `tsc --noEmit` 通过
- [x] 手动：登录 → 浏览器有 cookie → 消息/会话/好友/设置/实时全部可用
- [x] 手动：跨源请求被 CORS 拦截；WS 跨源被拒
- [x] 手动：登出 → cookie 清除 → 受保护路由 401

---

## Risky Files / Rollback Points

| File | Risk | Rollback |
|------|------|----------|
| `router.go` | CORS 重写可能影响所有跨源请求 | 保留原 withCORS 作对比 |
| `auth_middleware.go` | 认证来源变更影响所有受保护路由 | 保留 header 路径作 fallback（临时） |
| `Session.tsx` | token 移除影响全前端 | 改动集中在 Session/http 层 |
| `api/*.ts` | token 参数移除，调用方多 | 批量改，编译器保护 |

## Validation Commands

```bash
# Backend
go build ./...
go test ./...

# Frontend
npx tsc --noEmit

# Manual
# 1. 登录 → DevTools 看 Cookie（HttpOnly, SameSite=Lax, Secure?）
# 2. 刷新 → 会话保持（cookie 自动带）
# 3. 打开 WS → 连接成功（无 ?token=）
# 4. 从 http://evil.com 发请求 → 被 CORS 拦截
# 5. 登出 → cookie 清除，401
```
