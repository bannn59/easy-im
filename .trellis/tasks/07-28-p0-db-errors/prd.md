# Postgres 迁移与 API 错误中间件

## Goal

落地路线图 **T1 / P0**：为 easy-im API 接入 **PostgreSQL**、**版本化迁移**、**统一错误 JSON + `request_id`**，并可选提供本地 Compose，供后续 **T2 注册登录** 串行接上。  
**不含** 注册/登录/会话/消息业务。

## Background

- 路线图：`.trellis/tasks/archive/2026-07/07-28-feature-dev-roadmap/research/roadmap.md`（单人串行 T1→T2）。
- 现状：`backend` 仅有 stdlib HTTP + `/healthz`；`migrations/` 空；无 DB、无统一错误体。
- Spec：`database-guidelines.md`、`error-handling.md`、`logging-guidelines.md`。
- 环境：本机有 Docker；无全局 `migrate`/`psql` 亦可：用 Docker 跑 Postgres + `go run` 迁移 CLI 或 migrate 镜像。

## Scope

### In scope

1. **配置**：`DATABASE_URL`（或等价拆分）；缺省时行为明确（见 design：dev 可无 DB 启动 vs 强制——推荐 **API 可无 DB 启动 healthz**，但迁移与「DB 就绪」检查分开）。
2. **DB 连接**：`pgx` pool（或 `database/sql` + pgx stdlib）；进程关闭时释放。
3. **迁移**：`backend/migrations/` 使用 **golang-migrate** 或 **goose**（design 定一种）；至少一条 **基线迁移**（可含 `schema_migrations` 之外的扩展：`users` 表草案 **仅当** 为 T2 铺路且字段最小——见下）。
4. **`users` 最小表（推荐纳入 T1）**：`id`、`email`（或 username）、`password_hash`、`created_at` 等，**无** 注册 API；避免 T2 再抢迁移所有权。
5. **领域/应用错误**：`internal/apperr` 或 `internal/domain` 中稳定 `Code` + 安全 `Message`；sentinel 与 HTTP 映射表。
6. **中间件**：生成/传播 `request_id`（接受 `X-Request-ID`）；错误写入统一 JSON：
   ```json
   {"error":{"code":"...","message":"...","request_id":"..."}}
   ```
7. **演示路由（可选但建议）**：`GET /v1/boom` 或仅测试内触发，证明 500/映射不泄漏 driver 细节；或 health 保持简洁、单测覆盖映射。
8. **文档**：`backend/README` 更新：如何起 Postgres、跑迁移、跑 API。
9. **Compose（推荐）**：根或 `backend/docker-compose.yml` 仅 Postgres（+ 可选说明）。

### Out of scope

- 注册/登录 handler、JWT、前端登录页（**T2**）。
- Redis、MQ、gateway。
- sqlc 全量、ORM、多环境复杂配置中心。
- 强制 CI 里起 Docker（有则加；无则文档 + 单测不依赖真库为主）。

## Constraints

- 依赖方向：handler 映射错误；**不**在 service 里写 HTTP status（即使 service 尚未出现，apperr 也不应依赖 `net/http`）。
- 不向客户端返回 pg 英文原声。
- 迁移只增不改已发布文件（本仓库首次迁移除外）。
- 单人串行：本任务完成后才开 T2。

## Acceptance Criteria

- [x] `DATABASE_URL`（或文档等价）可配置；README 写明。
- [x] 使用 Docker（或文档中的外部 PG）可执行 **migrate up** 成功。
- [x] `migrations/` 至少 1 个版本化迁移；若含 `users`，字段满足 T2 最小注册需求且无业务 API。
- [x] API 进程能在「有 DB」下启动并仍提供 `GET /healthz` 200；可选 `GET /readyz` 检查 DB（若实现则写入 README）。
- [x] 统一错误 JSON 含 `code`、`message`、`request_id`；中间件/辅助函数有单测。
- [x] `request_id`：无入站头则生成；有则透传（到响应头或 body，design 定一种并一致）。
- [x] `go test ./...` 通过；不要求集成测试连真库（有则更好）。
- [x] 无注册登录产品功能伪装完成；无密钥进库。

## Notes

- 复杂任务：`design.md` + `implement.md` + jsonl，确认后 `task.py start`。
