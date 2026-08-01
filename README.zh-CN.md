# easy-im

> **[English](README.md)** · 简体中文

可自托管的即时通讯 monorepo。Go 后端（标准库 HTTP + PostgreSQL + Kafka）配合 React/TypeScript 前端。

| 目录 | 技术栈 | 状态 |
|------|--------|------|
| `backend/` | Go 1.25、标准库 HTTP、PostgreSQL（pgx）、Kafka（franz-go）、Web Push | 活跃 — 聊天、群组、实时、搜索、推送 |
| `frontend/` | Vite + React 18 + TypeScript | 活跃 — 工作区、聊天、搜索、PWA 推送 |
| `.trellis/` | Trellis 工作流 + 编码规范 | 活跃 |

## 功能

- **账号**：注册 / 登录 / 退出（HttpOnly Cookie 会话）、个人资料（显示名）、修改密码、中英双语 i18n。
- **会话**：1:1（从好友发起）与群聊；会话列表带预览、未读角标与时间。
- **群组**：建群、拉人 / 踢人 / 退群、转让群主、重命名群聊 —— 全部经 WebSocket 实时广播。
- **消息**：发送（`client_msg_id` 幂等）、历史分页、回复、编辑、撤回、已读回执、输入中提示、表情。
- **实时**：应用级单 WebSocket。多节点经 Kafka（`im.messages` / `im.presence`）扇出 —— 消息与群组事件可到达任意 API 节点上的成员。
- **搜索**：会话内搜索（可跳转到消息），以及全局跨会话搜索（ACL 限定）+ 关键词高亮。
- **推送**：worker 离线 Web Push 投递；PWA service worker + 推送设置开关。
- **可观测性**：Prometheus `/metrics`、结构化 JSON 日志、请求 ID、统一错误格式。

## 架构

```
┌──────────┐  HTTP / WS   ┌───────────────────────┐      ┌──────────────┐
│ 前端      │ ───────────► │  cmd/api（每节点）     │ ───► │ PostgreSQL   │
│ (React)  │              │  REST + WS hub + 认证  │      └──────────────┘
└──────────┘              │  fanout consumer (MQ) │
                          └──────────┬────────────┘
                                     │ Kafka: im.messages / im.presence
                          ┌──────────▼────────────┐
                          │  cmd/worker            │  离线 Web Push
                          └───────────────────────┘
```

- **传输**：写入/查询走 HTTP JSON API；实时推送与输入中命令走 WS（`/v1/ws`）。多节点投递由每节点 Kafka fanout consumer 完成（origin-skip 避免重复投递）。
- **主存储**：PostgreSQL（goose 迁移）。**异步总线**：Kafka 用于跨节点扇出与离线推送。

## 快速开始

需要 Docker（Postgres + Kafka）以及 Go ≥1.25 + Node ≥18。

### 1. 基础设施

```bash
docker compose up -d
```

### 2. 迁移

```bash
export DATABASE_URL='postgres://easyim:easyim@localhost:5433/easyim?sslmode=disable'
cd backend && go run ./cmd/migrate up
```

### 3. API

```bash
cd backend
export DATABASE_URL='postgres://easyim:easyim@localhost:5433/easyim?sslmode=disable'
export AUTH_JWT_SECRET='easyim-dev-secret-change-me'   # 开发密钥；其他环境请设置真实密钥
export KAFKA_BROKERS='localhost:19092'                  # 可选；多节点实时与推送
go run ./cmd/api
# GET http://localhost:8080/healthz  → {"status":"ok"}
# GET http://localhost:8080/readyz   → {"status":"ok"}（DB 已就绪）
```

可选环境变量：`PORT`（默认 8080）、`CORS_ALLOWED_ORIGINS`、`METRICS_ADDR`（Prometheus）、`COOKIE_SECURE`、`COOKIE_DOMAIN`。

### 4. Worker（离线推送）

```bash
cd backend
DATABASE_URL="$DATABASE_URL" KAFKA_BROKERS='localhost:19092' \
  VAPID_PUBLIC_KEY=... VAPID_PRIVATE_KEY=... PUSH_SUBJECT=mailto:you@example.com \
  go run ./cmd/worker
```

推送是可选的。没有 VAPID 密钥 API 仍可运行，仅离线推送被禁用。

### 5. Web

```bash
cd frontend
npm install
npm run dev
# http://localhost:5173
```

可选：若 API 不在 `http://localhost:8080`，复制 `frontend/.env.example` 到 `frontend/.env` 并设置 `VITE_API_BASE`。

## 测试 / 构建

```bash
cd backend && go test ./...
cd frontend && npm run typecheck && npm run build
```

## API 概览（节选）

| 方法 | 路径 | 用途 |
|------|------|------|
| `POST` | `/v1/auth/register` `/v1/auth/login` `/v1/auth/logout` | 认证（Cookie 会话） |
| `GET/PATCH` | `/v1/me`、`/v1/me/profile` | 会话、个人资料 |
| `POST` | `/v1/me/password` | 修改密码 |
| `GET` | `/v1/conversations` | 会话列表 |
| `POST` | `/v1/conversations/groups` | 创建群聊 |
| `GET` | `/v1/conversations/{id}` | 会话详情 |
| `POST` | `/v1/conversations/{id}/messages` | 发送消息 |
| `GET` | `/v1/conversations/{id}/messages` | 历史 / `around_seq` 窗口 |
| `GET` | `/v1/conversations/{id}/messages/search` | 会话内搜索 |
| `PATCH` | `/v1/conversations/{id}` | 重命名群聊（群主） |
| `POST/DELETE` | `/v1/conversations/{id}/members*`、`/owner` | 成员管理 |
| `GET` | `/v1/search/messages` | 全局搜索（ACL 限定） |
| `GET` | `/v1/friends*` | 好友与请求 |
| `GET` | `/v1/ws` | WebSocket（实时） |
| `GET` | `/metrics` | Prometheus |

## 规范

编码约定位于 `.trellis/spec/`（backend、frontend、guides）。它们基于源码，随真实模式落地而更新。

## Agents

面向 Trellis 助手的说明见 `AGENTS.md`。
