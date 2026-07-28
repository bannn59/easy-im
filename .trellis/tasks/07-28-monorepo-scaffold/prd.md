# 脚手架 monorepo backend+frontend

## Goal

为 **easy-im** 落下可编译/可启动的 monorepo 骨架：

- `backend/`：Go module + `cmd/api` 健康检查 HTTP 服务  
- `frontend/`：Vite + React + TypeScript 空壳与最小路由  

对齐 `.trellis/spec` bootstrap 目录约定。**不含**完整 IM 业务（登录、会话、消息、WS gateway 等）。

## Background

- 功能地图调研结论：0 个已实现用户功能；无 `backend/`/`frontend/`。
- Spec 已约定 Go monorepo + React/TS；本任务把「目标路径」变成真实目录与最小可运行入口。
- 环境已具备：Go 1.26、Node 18、npm 9。

## Scope

### In scope

1. `backend/go.mod`（module path 合理，如 `github.com/ban/easy-im/backend` 或 `easy-im/backend`——以本地无远程为准选 `easy-im/backend`）。
2. 最小分层占位，与 spec 一致且不过度：
   - `cmd/api` 可启动
   - `internal/handler` 健康检查
   - `internal/config` 读 `PORT`（默认 8080）
   - 可选空目录或 README 说明 `cmd/gateway`、`cmd/worker` 占位（**不**实现 WS/MQ）
3. `GET /healthz`（或 `/health`）返回 JSON：`{"status":"ok"}`（可含 version 占位）。
4. `frontend/`：Vite React-TS；目录含 `src/app`（路由）、`src/api`、`src/realtime` 占位、`src/features` 可空、`src/shared`。
5. 前端首页可渲染；可调用或展示 API base URL 配置占位（不必真连后端 CI）。
6. 根或包级 README 简短说明如何 `go run` / `npm run dev`。
7. `.gitignore` 已覆盖的依赖目录不提交；`package-lock.json` 可提交。
8. 能通过：`cd backend && go test ./...`（至少编译相关包）；`cd frontend && npm run build`（或 typecheck+build）。

### Out of scope

- 真实鉴权、Postgres、Redis、MQ、WebSocket gateway 实现  
- `packages/contracts` 完整 OpenAPI 生成管线（可只留空目录或一句话 README）  
- Docker Compose / K8s  
- 改写大段 `.trellis/spec`（仅当路径与脚手架不一致时做**最小** bootstrap 状态更新，可选）  
- UI 设计系统定稿（可用极简默认样式）

## Constraints

- 依赖方向遵守 spec：`handler` 不直连未来的 `repo`；健康检查可放 handler 或极薄 service。
- 前端禁止在组件里 `new WebSocket`；`realtime/` 可先空模块 + 注释。
- 不引入重型 ORM / 状态库；前端可不装 TanStack Query（首屏无服务端列表时），但 `api/` 目录要在。
- 中文 PRD；代码标识与注释以英文为主，与现有 spec 一致。

## Acceptance Criteria

- [x] 存在 `backend/go.mod`，`cd backend && go build ./cmd/api` 成功。
- [x] `go run ./cmd/api`（或等价）监听端口，`GET /healthz` 返回 200 + JSON status ok。
- [x] 存在与 spec 对齐的 `internal/` 基本包（至少 `handler`、`config`；`domain` 可有空/错误占位）。
- [x] 存在 `frontend/package.json`，`npm install` + `npm run build` 成功。
- [x] 前端为 React+TS+Vite；含 `src/app` 路由与至少一页可访问 UI。
- [x] 有 `src/api`、`src/realtime` 占位，无业务 IM 功能伪装成已完成。
- [x] 文档说明本地启动 backend / frontend 的命令。
- [x] 无密钥提交；工作区可提交文件均为脚手架与文档。

## Notes

- 复杂任务：需 `design.md` + `implement.md`，评审后再 `task.py start`。
- 完成后建议新开任务做「健康检查联调」或「鉴权最小闭环」，并刷新功能地图。
