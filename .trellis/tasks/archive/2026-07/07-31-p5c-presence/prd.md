# P5.c Online Presence

## Goal

Expose online/offline status for users so friends can see who is available, without treating presence as an ACL or message-history source of truth.

## Background

P5.c was tagged **High risk** in the roadmap. The in-process hub already tracks live connections internally (`userID → set of *Client`), but there is no product-facing presence API. The spec (realtime-messaging.md) states presence is **ephemeral** and must NOT become an ACL or message-history truth source.

### Current technical foundation (pending research confirmation)

- Hub tracks online connections by user ID (multi-device)
- No `last_seen` column on users table (pending confirmation)
- Frontend has no online-status UI yet

## Scope (confirmed)

P5.c Online presence. Other P5 items deferred.

## Confirmed Facts

- **Hub** 已通过 `clients map[userID]set[*Client]` 隐式追踪在线状态（`len(clients[uid]) > 0` = 在线），但没有任何产品层接口暴露它
- **无 `last_seen`**：users 表、`domain.User`、`UserRepo` 都没有 last_seen 字段/列，需要 migration + repo 方法
- **WS dispatcher** 有干净扩展点：`HandleFrame` switch + `IsMember → ListMemberIDs → BroadcastToConversation` 模式（typing 已示范）
- **Spec 约束**：presence 是 ephemeral（Redis），online = ≥1 healthy conn，DB last_seen 仅异步；查询走 HTTP，fan-out 主题 `im.presence.changed`；不得成为 ACL 或消息投递机制；需限流
- **前端 UI 自然位置**：FriendsPage 好友行、ConversationRoom header（DM 显示 peer，群聊显示成员列表）
- **数据结构**：`PublicUser { id, email }` 和 `Conversation.members` 需要新增 online/last_seen 字段
- **无心跳机制**：当前 WS 无 ping/pong，连接活着 = 在线（浏览器 onclose 通常很快触发）

## Scope (confirmed)

- **在线/离线圆点**：好友列表 + 会话头显示在线状态圆点
- 不做 `last_seen` 时间戳持久化（推迟到后续迭代）
- 广播范围：**好友全局广播**（某人上线/下线 → 推送 `presence.changed` 给其所有好友）

## Requirements

### R1 — Presence 状态推导

- 在线 = hub 中该用户有 ≥1 个健康 WS 连接（`len(clients[uid]) > 0`）
- 无心跳：连接活着 = 在线；断开 = 离线

### R2 — HTTP 查询

- `GET /v1/friends` 返回的好友列表包含每个好友的 `online` 布尔字段
- 初始加载时基于当前 hub 连接状态计算

### R3 — WS 实时广播

- 用户上线/下线时，hub 广播 `presence.changed` 事件给该用户的所有好友
- 事件 payload: `{ user_id, online }`
- 多设备：任一设备连接 → 在线；所有设备断开 → 离线（只广播一次状态变化，不按连接数重复推送）

### R4 — 前端 UI

- **好友列表页**：每个好友行显示在线/离线圆点
- **会话头**：DM 会话显示对方在线/离线圆点
- 圆点样式遵循极简黑白灰设计语言（在线=深色实心点，离线=空心/浅色点）

## Acceptance Criteria

- [ ] 两个好友 A、B。A 登录（WS 连接建立）后，B 的好友列表页看到 A 在线圆点（无需刷新）
- [ ] A 关闭浏览器/断开 WS，B 的好友列表 A 变为离线圆点（无需刷新）
- [ ] `GET /v1/friends` 响应包含每个好友的 `online` 字段
- [ ] DM 会话头显示对方在线/离线圆点，随 `presence.changed` 实时更新
- [ ] 多设备：A 同时开两个标签页，关掉一个仍显示在线；全部关闭才变离线
- [ ] 群聊会话头不显示"群成员在线"（本轮不做）
- [ ] 未知/重复的 presence 事件不导致前端崩溃或重复渲染
- [ ] 无 last_seen 持久化（数据库无变更）

## Out of Scope

- Presence as ACL / message-history source of truth (roadmap prohibition)
- `last_seen` 时间戳持久化（DB 无变更，推迟到后续迭代）
- Multi-node / Redis presence fan-out (P6)
- Offline push notifications (P6)
- 群聊会话头的"成员在线"指示（本轮不做）
- 心跳 / 空闲检测（连接活着即在线）
- 在线状态限流（spec 建议，但本轮连接数规模小）

## Open Questions

_(none — all blocking decisions resolved)_
