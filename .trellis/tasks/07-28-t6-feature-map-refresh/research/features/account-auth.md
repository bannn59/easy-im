# 账号与鉴权

| 功能 | status | entry | code | tests |
|------|--------|-------|------|-------|
| 注册 | implemented | POST `/v1/auth/register`, `/register` | `auth_service.go`, `AuthPage` | service unit + smoke |
| 登录 | implemented | POST `/v1/auth/login`, `/login` | 同上 | 同上 |
| 当前用户 | implemented | GET `/v1/me` | `AuthHandler.Me` | smoke |
| 会话保持 | implemented | localStorage token + SessionProvider | `Session.tsx` | build |

deps: `users` 表, `AUTH_JWT_SECRET`, bcrypt, JWT HS256  
risk: medium (token storage)
