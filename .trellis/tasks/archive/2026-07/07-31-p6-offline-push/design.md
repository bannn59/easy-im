# P6 Offline push (Web Push PWA) — Design

## Architecture

```
┌─────────────┐   msg.created(HTTP)   ┌──────────────┐
│  cmd/api     │ ──produce──────────▶ │   Kafka      │
│  (HTTP+WS)   │                      │              │
│  hub in mem  │ ──presence.event────▶│  presence.   │
└─────────────┘   (online/offline)    │  events topic │
                                      └──────┬───────┘
                                             │ consume
                                      ┌──────▼───────┐
                                      │  cmd/worker  │
                                      │ local online │  fetch members + subs
                                      │ set (from    │  from Postgres
                                      │ presence)    │
                                      └──────┬───────┘
                                             │ send Web Push (VAPID)
                                      ┌──────▼───────┐
                                      │ Browser SW  │
                                      └─────────────┘
```

### Data flow (message send)

1. `POST /v1/conversations/{id}/messages` → `MessageService.Send` 落库成功。
2. API 进程 produce `msg.created` 事件到 Kafka topic `im.messages`（key = `conversationID`，保证会话内顺序）。**现有 `broadcast`（内存 WS 扇出）保持不变**——在线用户仍走 hub 即时收到。
3. API 进程同时广播 `presence.changed` 到 Kafka topic `im.presence`（hub 在线/离线 transition 时），key = `userID`。
4. worker 消费 `im.messages`：查会话成员 → 对每个成员查本地在线集合（由消费 `im.presence` 维护）→ 离线成员查 `push_subscriptions` 表 → 发送 Web Push（带聚合窗口）。
5. worker 消费 `im.presence`：更新本地 `onlineUsers map[string]bool`（replay 自 Kafka 或先全量拉 DB 兜底）。

### 离线判定（跨进程的关键）

- worker 不查询 DB 里的“最后活跃”，而是**维护从 Kafka presence topic 学到的在线集合**。
- 启动时先消费历史 presence 事件建立基线（Kafka offset 从最早开始），再增量更新。多 api 进程时天然正确（每个进程都发布 presence 事件）。
- 单 api 进程部署下：api 发布 presence 事件到 Kafka → worker 消费 → 离线判定与 hub 一致。

### 聚合窗口（会话聚合）

- worker 对同一 `(conversationID)` 的推送做**时间窗聚合**（如 2s / 消息数上限 10）：窗口内多条消息合并为一条通知「N 条新消息」+ 最新消息预览。
- 实现：`internal/push` 包的 `Aggregator`，按 conversationID 分桶，计时器触发发送。窗口到点后查询最新消息做预览。
- 通知带 `tag: conversationID`，浏览器同 tag 替换旧通知，不堆叠。

## Components

### Backend

| Package | Responsibility |
|---------|----------------|
| `internal/push` | Web Push 发送核心：VAPID 签名、payload 加密、HTTP 发送、410/404 失效清理。用 `github.com/wuc656/webpush-go` |
| `internal/push/subscription_repo.go` | `push_subscriptions` 表 CRUD（按用户、按 endpoint） |
| `internal/mq` | franz-go 封装：producer、consumer group（`msg.created` + `presence.changed`）。遵循 spec `directory-structure.md` 的 `internal/mq` 约定 |
| `internal/service/message_service.go` | Send 成功后 produce `msg.created` 到 Kafka（新增一个 `EventProducer` 依赖，nil-safe 保持现有测试可跑） |
| `internal/hub` | 新增 `PresencePublisher` 回调（在线/离线时发布到 Kafka presence topic） |
| `cmd/worker` | 新进程：消费 Kafka → 离线判定 → 聚合 → Web Push |
| `cmd/migrate` | 新增 `push_subscriptions` 表迁移 |
| `internal/handler/push.go` | `POST /v1/push/subscriptions`（注册）、`DELETE /v1/push/subscriptions`（注销） |
| `internal/config` | 新增 `KafkaBrokers`、`VAPIDPublicKey`、`VAPIDPrivateKey`、`PushSubject` |

### Frontend

| File | Responsibility |
|------|----------------|
| `public/sw.js` | Service Worker：`push` 事件 → 展示通知；`notificationclick` → 打开会话 |
| `public/manifest.webmanifest` | PWA manifest（name、icons、start_url） |
| `frontend/src/features/settings/PushSettings.tsx` | 设置页「推送通知」开关：请求权限 → `registration.pushManager.subscribe` → 调后端注册/注销 |
| `frontend/src/realtime` | 不变（WS 仍负责在线即时消息） |
| `frontend/vite.config.ts` | 无需改动（`public/` 自动托管 SW + manifest） |

## Contracts

### DB: `push_subscriptions`

```sql
CREATE TABLE push_subscriptions (
    id            UUID PRIMARY KEY,
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    endpoint      TEXT NOT NULL,
    p256dh        TEXT NOT NULL,
    auth          TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX uq_push_subs_user_endpoint ON push_subscriptions (user_id, endpoint);
```

### API

| Method | Path | Auth | Body / Note |
|--------|------|------|-------------|
| POST | `/v1/push/subscriptions` | cookie | `{endpoint, p256dh, auth}`；upsert 按 (user_id, endpoint) |
| DELETE | `/v1/push/subscriptions` | cookie | body `{endpoint}`；删除 |

### Kafka topics

| Topic | Key | Value (JSON) |
|-------|-----|--------------|
| `im.messages` | conversationID | `{id, conversation_id, sender_id, body, created_at}`（message DTO） |
| `im.presence` | userID | `{user_id, online, at}` |

### Web Push payload（worker → SW）

```json
{
  "type": "chat_message",
  "title": "<sender display name>",
  "body": "<preview>（或「N 条新消息」）",
  "conversation_id": "<id>",
  "tag": "<conversation_id>",
  "count": 3
}
```

## Config（env）

| Env | Purpose |
|-----|---------|
| `KAFKA_BROKERS` | 逗号分隔 broker 列表，默认 `localhost:9092` |
| `VAPID_PUBLIC_KEY` | Web Push VAPID 公钥 |
| `VAPID_PRIVATE_KEY` | VAPID 私钥（dev 可 `VAPID_DEV_INSECURE=1` 生成/固定 dev 值） |
| `PUSH_SUBJECT` | VAPID subject（mailto:） |
| `PUSH_AGGREGATE_WINDOW` | 聚合窗口，默认 2s |

## Trade-offs

- **在线状态经 Kafka 而非 DB**：worker 不查 DB 判在线，避免 DB 轮询，且多 api 进程天然一致；代价是 presence 事件有秒级延迟（可接受）。
- **Kafka 是重依赖**：本地 docker-compose 需加 `kafka`（+KRaft，单 broker）服务。生产形态对齐 OpenIM 式扇出。
- **Worker 与 API 共享同一套 repo**：worker 直接用 `repo` 包查成员/订阅/预览，无需重复实现。
- **聚合窗口在 worker**：与 OpenIM「toOfflinePush 消费组独立聚合」一致。

## Rollback

- 停 worker 进程 → 功能退化为「无离线推送」（在线 WS 不受影响）。无数据破坏。
- 移除 `push_subscriptions` 迁移可回滚（worker 停止后）。
- Kafka 缺失时：api 的 produce 应 nil-safe/降级（不阻塞消息发送主路径）。

## Open items to confirm during implementation

- franz-go producer/consumer 的具体 API 形态（seed / manual commit）。
- webpush-go 的订阅结构体与 VAPID 选项。
- docker-compose 中 Kafka 单 broker 的 KRaft 最小配置。
