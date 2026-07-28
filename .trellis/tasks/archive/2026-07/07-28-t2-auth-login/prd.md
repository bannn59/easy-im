# 用户注册登录与会话保持

## Goal

落地路线图 **T2 / P1 / M1**：用户可 **注册、登录**，前端在刷新后仍保持会话，并看到当前用户（`/me` + 主框）。  
**依赖** 已归档 T1：`users` 表、pgx、`apperr`、统一错误 JSON。  
**不含** 会话列表、消息、OAuth、刷新令牌轮换、WS ticket（P1.3 可延后）。

## Background

- 路线图：`.trellis/tasks/archive/2026-07/07-28-feature-dev-roadmap/research/roadmap.md`
- T1 归档：`.trellis/tasks/archive/2026-07/07-28-p0-db-errors/`
- 表：`users (id UUID PK, email UNIQUE, password_hash, created_at, updated_at)`
- 前端：Vite React，仅 Home / Health；`api/client.ts` 仅 healthz

## Scope

### In scope

**Backend**

1. `POST /v1/auth/register` — email + password → 创建用户，返回 user + access token（或 201 + 需再 login，**推荐直接发 token** 减少往返）。
2. `POST /v1/auth/login` — email + password → access token + user。
3. `GET /v1/me` — `Authorization: Bearer <token>` → 当前用户；无/坏 token → 401 `unauthorized`。
4. 密码：**bcrypt**（或 argon2；design 定一种）哈希存 `password_hash`；永不回传 hash。
5. Token：**JWT**（HS256）或不透明随机 token 存内存/表；MVP 推荐 **JWT + `AUTH_JWT_SECRET`**，短过期（如 7d dev / 可配置）。
6. 分层：`internal/repo` user、`internal/service` auth、`internal/handler` auth；**不**在 handler 写 SQL。
7. 校验：email 格式粗检、password 最小长度（如 ≥8）；冲突 email → 409 `conflict`。
8. CORS：允许 `Authorization` 头（及前端 dev origin 现状 `*` 可保留并注明）。
9. **必须** `DATABASE_URL` 配置才能 register/login；未配置时这些路由 503 `unavailable`。

**Frontend**

1. `/login`、`/register` 页面（可合并为一带 tab 的 auth 页）。
2. Session：内存 + `localStorage` 存 access token（MVP；威胁模型写明 XSS 风险，后续可改 httpOnly cookie）。
3. 登录成功后进入「主框」：显示 email/id、退出；未登录访问受保护路由 → 重定向 login。
4. `api` 封装：register/login/me；401 清 session。
5. 导航：Login / 用户区；Health 可仍公开。

### Out of scope

- Refresh token、登出服务端黑名单、邮箱验证、密码重置  
- OAuth、2FA、RBAC  
- 会话/消息、WS ticket  
- 生产级 cookie 方案、CSRF  

## Constraints

- 单人串行：基于 T1，不改迁移已应用文件（改 schema 需 **新** goose 版本）。
- 错误走现有 `WriteError` / `apperr`；不泄漏「email 是否存在」以外的必要信息时：login 统一 `unauthorized`（不区分用户不存在 vs 密码错）——**推荐**防枚举。
- `apperr` 仍不 import `net/http`。
- 密码与 JWT secret 仅环境变量；compose/README 给 **dev 默认 secret** 并警告。

## Acceptance Criteria

- [x] `POST /v1/auth/register` 成功创建 `users` 行并返回 token + 安全 user 字段（id, email）。
- [x] 重复 email → 409 + 稳定 code；弱密码/坏 email → 400。
- [x] `POST /v1/auth/login` 正确凭证发 token；错误凭证 401，不区分原因。
- [x] `GET /v1/me` 带 Bearer 返回当前用户；无 token/坏 token → 401。
- [x] 密码仅以 hash 存储；响应无 `password_hash`。
- [x] 前端可注册/登录，刷新后仍显示已登录用户；退出清除 token。
- [x] 受保护页（如 `/app` 或 `/` 主框）未登录会去登录页。
- [x] `go test` 覆盖 service/handler 关键路径（可用 testcontainers 或 repo 接口 fake）；`npm run build` 通过。
- [x] README 更新 auth 环境变量与示例 curl。
- [x] 无会话/消息功能伪装完成。

## Dependencies

- **Blocked by（已满足）**: T1 users 表、DB pool、apperr、migrate。
- **Blocks**: T3 会话（需 user id）。

## Notes

- 复杂任务：需 design + implement，确认后 `task.py start`。
