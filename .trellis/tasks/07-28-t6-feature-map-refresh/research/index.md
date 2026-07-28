# easy-im 功能地图（刷新 · 2026-07-28）

**任务**: `07-28-t6-feature-map-refresh`  
**相对**: 归档空地图 + 脚手架后实现 M0–M4

## 统计

| 指标 | 数量 |
|------|------|
| **implemented** 用户可感知能力组 | **5**（健康、鉴权、会话、HTTP 消息、WS 实时） |
| **partial** | 1（设置/资料仅 session 展示，无独立设置页） |
| **not_found / planned** | 已读回执、输入中、presence、离线推送、管理后台 |

## 导航

| 文档 | 说明 |
|------|------|
| [method.md](./method.md) | 扫描方法 |
| [features/](./features/) | 分组明细 |
| [notes/milestones.md](./notes/milestones.md) | M0–M5 与验证 |
| [notes/remaining.md](./notes/remaining.md) | P5/P6 未做项 |

## 分组总览

| 分组 | 状态 | 入口 | 关键代码 |
|------|------|------|----------|
| 基建 / 健康 | **implemented** | `/healthz`, `/readyz`, FE `/health` | `handler`, `HealthPage` |
| 账号与鉴权 | **implemented** | `/login` `/register` `/v1/auth/*` `/v1/me` | `service/auth_*`, `Session` |
| 会话 | **implemented** | `/app`, `POST/GET /v1/conversations` | `conversation_*`, `AppShell` |
| 消息（HTTP） | **implemented** | room composer, `/messages` | `message_*` |
| 实时（WS） | **implemented** | `/v1/ws`, FE `realtime/` | `hub`, `ws.go` |
| 回执 / 输入中 | not_found | — | — |
| 在线 presence | not_found（仅有 conn hub，无 presence API） | — | `hub` 内部 |
| 推送 / 设置 / 管理 | not_found / partial | — | — |

## 高风险 / 高耦合（现有代码）

| 主题 | 说明 |
|------|------|
| 消息发送 + 扇出 | HTTP insert 后同步 `PublishToUsers`；单节点 hub |
| WS 鉴权 | query `token` + dev `CheckOrigin: true` |
| localStorage JWT | XSS 即账号；MVP 已知债 |
| CORS `*` | 仅适合本地 |

## 适合新人

| 模块 | 原因 |
|------|------|
| `HealthPage` / healthz | 无业务状态 |
| 会话列表 UI 样式 | 边界清晰 |
| 设置页（若做 P5.d） | 少服务依赖 |

**不建议新人第一项**: `hub` 扇出、幂等 seq 事务、WS 重连。
