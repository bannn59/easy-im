# Design: P6 Production CORS & Auth Hardening

## Architecture Overview

```
[登录/注册] → AuthService 签发 JWT → handler 设 Set-Cookie(easyim_session, HttpOnly, SameSite=Lax, Secure?)
   ↑                                                                      ↓
[前端] 不再存 localStorage，请求自动带 cookie                    [受保护路由] require 中间件从 cookie 读 JWT

[WS] 同源连接自动带 cookie → 后端 r.Cookie 读 JWT → CheckOrigin 校验 Origin
```

- **cookie 存 JWT**（无状态，复用现有 ParseAccessToken）
- **CORS allowlist** 由 env 驱动，动态回显 + `Vary: Origin` + credentials
- **只认 cookie**，移除 Authorization header 路径

## 1. Config — 新增 env

**Files:** `backend/internal/config/config.go`

```go
type Config struct {
    ...
    CORSAllowedOrigins []string  // 新：CORS_ALLOWED_ORIGINS，逗号分隔
    CookieSecure      bool       // 新：COOKIE_SECURE（生产 true，dev false）
    CookieDomain      string     // 新：COOKIE_DOMAIN（可选）
}
```

- `CORS_ALLOWED_ORIGINS`: 逗号分隔，空 = 默认 `http://localhost:5173`（dev 前端）？需考虑 dev 便捷
- `COOKIE_SECURE`: 默认 false（dev），生产设 true
- 启动校验：`AUTH_DEV_INSECURE` 未设 且 `AUTH_JWT_SECRET` 空 → `log.Fatal`（拒绝启动）

## 2. Cookie 常量

**Files:** `backend/internal/handler/auth.go` 或新 `cookies.go`

```go
const sessionCookieName = "easyim_session"

// CookieConfig holds cookie attributes for the session cookie.
type CookieConfig struct {
    Secure bool
    Domain string
}
```

## 3. 认证来源：header → cookie

### AuthHandler 签名变更

`Login` / `Register` 成功后：
```go
http.SetCookie(w, &http.Cookie{
    Name:     sessionCookieName,
    Value:    res.AccessToken,
    Path:     "/",
    HttpOnly: true,
    SameSite: http.SameSiteLaxMode,
    Secure:   cfg.Secure,
    Domain:   cfg.Domain, // 空 = 当前域
})
// 响应体不再返回 access_token（或保留 token_type/user 但去掉 token）
```

新增 `Logout` handler：
```go
// POST /v1/auth/logout — 清除 cookie
http.SetCookie(w, &http.Cookie{Name: sessionCookieName, Value: "", Path: "/", MaxAge: -1, HttpOnly: true, SameSite: http.SameSiteLaxMode, Secure: cfg.Secure})
```

### require 中间件

`RequireUser` 改为从 cookie 读：
```go
cookie, err := r.Cookie(sessionCookieName)
if err != nil { return 401 }
token := cookie.Value
// 其余不变：ParseAccessToken → 注入 context
```

### WS handler

`ws.go` 同样从 cookie 读 token：
```go
cookie, err := r.Cookie(sessionCookieName)
// 不再读 query string / Authorization header
```

## 4. CORS 中间件重写

**Files:** `backend/internal/handler/router.go`

```go
func withCORS(allowed map[string]struct{}) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            origin := r.Header.Get("Origin")
            if origin != "" {
                if _, ok := allowed[origin]; ok {
                    w.Header().Set("Access-Control-Allow-Origin", origin)
                    w.Header().Set("Access-Control-Allow-Credentials", "true")
                    w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
                    w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Request-ID")
                    w.Header().Add("Vary", "Origin")
                    if r.Method == http.MethodOptions {
                        w.WriteHeader(http.StatusNoContent)
                        return
                    }
                } else {
                    // 不在白名单：设 Vary 但不设 CORS 头 → 浏览器拦截
                    w.Header().Add("Vary", "Origin")
                }
            }
            next.ServeHTTP(w, r)
        })
    }
}
```

- 白名单来自 config `CORSAllowedOrigins`
- 移除了 `Authorization` from allowed headers（不再用 header 认证）
- 非浏览器请求（无 Origin，如 curl 同源）不受影响

## 5. WS CheckOrigin

**Files:** `backend/internal/handler/ws.go`

```go
upgrader := websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        origin := r.Header.Get("Origin")
        if origin == "" { return true } // 非浏览器客户端（无 Origin）允许
        _, ok := allowedOrigins[origin]
        return ok
    },
}
```

- `allowedOrigins` 从 config 传入 WSHandler

## 6. 前端改造

### 不再存 localStorage

`Session.tsx`：
- 删除 `TOKEN_KEY` 和 localStorage 读写
- token 从哪来？**cookie 是 HttpOnly，前端 JS 读不到**。所以前端无法直接拿到 token 字符串。
- **方案**：Session 只跟踪 `user`（登录状态），不发 token 到 API 调用。所有 API 请求改为 `credentials: 'include'`，让浏览器自动带 cookie。

`api/http.ts`：
```typescript
fetch(url, { credentials: 'include', ... })
```
- `token` 参数从所有 API 函数移除？—— 改动大。**替代**：保留 token 参数但忽略（兼容），或逐步移除。设计上建议移除。
- WS：`connectRealtime` 不再拼 `?token=`，直接连同源 `ws://.../v1/ws`（cookie 自动带）

### Session 状态

`SessionState` 的 `token` 字段：
- 前端不再需要 token 字符串 → 移除 `token`，只用 `user` 判断登录态
- `login`/`register`：调 API，成功后 cookie 已设置，再 `fetchMe` 拿 user
- `logout`：调 `POST /v1/auth/logout` 清 cookie，再清 user state

### 影响面

前端所有 `session.token` 使用点都要改（AppShell、ConversationRoom、FriendsPage、SettingsPage、api/*）。`api/*` 的 `token` 参数改为 `credentials: 'include'`。

## 7. 数据流

### 登录
```
POST /v1/auth/login {email, password}
  → AuthService.Login → JWT
  → handler Set-Cookie(easyim_session=JWT, HttpOnly, SameSite=Lax, Secure?)
  → 响应 {user: {...}}（无 access_token）
  → 前端 fetchMe → Session.user
```

### 受保护请求
```
浏览器自动带 Cookie: easyim_session=JWT
  → require 中间件 r.Cookie → ParseAccessToken → 注入 uid
```

### WS
```
同源 ws://.../v1/ws → 浏览器自动带 cookie
  → WSHandler r.Cookie → ParseAccessToken → CheckOrigin 校验
```

### 登出
```
POST /v1/auth/logout → Set-Cookie(easyim_session=, MaxAge=-1) 清除
  → 前端清 user state
```

## 8. 兼容性 / 回滚

- **API 变更**：`/v1/auth/login|register` 响应去掉 `access_token`（breaking，前端同步改）；`Authorization` header 认证路径移除（dev 工具需改）
- **前端**：所有 `session.token` 使用点移除 token 传递，改 `credentials: 'include'`
- **回滚**：恢复 localStorage + header 认证（改动集中在 Session/http 层，回滚可控）
- **保留**：`AUTH_DEV_INSECURE=1` dev secret（文档化）；TTL 24h 生效
