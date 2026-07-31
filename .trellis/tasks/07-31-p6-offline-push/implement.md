# P6 Offline push (Web Push PWA) — Implement

## Ordered checklist

### Backend — infra & schema

- [x] **B1** 新增 migration `backend/migrations/20260731200000_push_subscriptions.sql`（`push_subscriptions` 表 + `uq_push_subs_user_endpoint`）。验证：`cd backend && go run ./cmd/migrate up` 无错；`psql \d push_subscriptions`。
- [x] **B2** `internal/repo/push_subscription_repo.go`：`Upsert(ctx, userID, endpoint, p256dh, auth)`、`ListByUser(ctx, userID)`、`DeleteByEndpoint(ctx, userID, endpoint)`、`DeleteByEndpoints(ctx, []endpoint)`（失效清理用）。验证：repo 单测。

### Backend — Kafka（franz-go）

- [x] **B3** `internal/mq/kafka.go`：franz-go producer/consumer 封装（遵循 spec `internal/mq` 约定）。
  - Producer：`Publish(topic, key, value)`；**nil-safe / 可注入假实现**，避免 api 在 Kafka 未配置时阻塞消息主路径。
  - Consumer：consumer group（`im.messages` / `im.presence`）、at-least-once（处理后提交 offset）。
  - 验证：`go build ./internal/kafka`。
- [x] **B4** `internal/config` 新增 `KafkaBrokers`、`VAPIDPublicKey`、`VAPIDPrivateKey`、`PushSubject`、`PushAggregateWindow`。验证：config 单测。

### Backend — push 核心

- [x] **B5** `internal/push/push.go`：用 `github.com/wuc656/webpush-go` 实现 `Send(sub, payload)`——VAPID 签名、aes128gcm 加密、HTTP 发送。返回结果区分：`delivered` / `gone`（410/404，需清理订阅）/ `error`。
- [x] **B6** `internal/push/aggregator.go`：会话聚合器。按 `conversationID` 分桶，`PushAggregateWindow`（默认 2s）到点批量发送，`tag: conversationID`。验证：聚合器单测（窗口触发、计数、预览取最新）。

### Backend — api 侧接线

- [x] **B7** `internal/handler/push.go`：`POST /v1/push/subscriptions`（cookie 鉴权，upsert）、`DELETE /v1/push/subscriptions`（注销）。注册到 `router.go`。
- [x] **B8** `MessageService.Send`：成功落库后 produce `msg.created` 到 Kafka topic `im.messages`（新增 `EventProducer` 依赖，nil-safe，不影响现有测试）。
- [x] **B9** hub presence 事件发布到 Kafka `im.presence`（`PresencePublisher` 回调，nil-safe）。验证：现有 presence 测试不破坏。

### Backend — worker

- [x] **B10** `cmd/worker/main.go`：新进程。消费 `im.presence` 维护本地在线集合；消费 `im.messages` → 查会话成员 → 离线成员查订阅 → 聚合发送。
- [x] **B11** docker-compose 加 `kafka` 服务（KRaft 单 broker，`localhost:9092`）。验证：`docker compose up -d kafka` + `docker exec` 查 topic。

### Frontend — PWA

- [x] **F1** `public/sw.js`：`push` → 通知（title/body/tag）；`notificationclick` → 打开 `/c/{conversation_id}` 并 close；`message.created` 前台不打扰（由 app 判断在线）。
- [x] **F2** `public/manifest.webmanifest` + `index.html` 加 manifest link + SW 注册。
- [x] **F3** `frontend/src/features/settings/PushSettings.tsx`：设置页开关。开启 → 请求权限 → `pushManager.subscribe`（VAPID 公钥）→ POST 订阅；关闭 → DELETE 订阅。接入 `/settings` 路由。验证：`npm run typecheck`。

### 收尾

- [x] **C1** 全量测试：`cd backend && go vet ./... && go test ./...`；`cd frontend && npm run typecheck && npm run build`。
- [x] **C2** 手动端到端：两浏览器 A/B → A 发消息 → B 关闭页面仍收到系统通知 → 点击打开会话。在线时不打扰。
- [x] **C3** spec 更新：`.trellis/spec/backend/realtime-messaging.md`（进程边界：`cmd/worker` landed；推送事件流）、`.trellis/spec/backend/database-guidelines.md`（`push_subscriptions` 表）。可选：新增 `push-push.md` 或并入 realtime。

## Validation commands

```bash
cd backend && go vet ./... && go test ./...
cd backend && go run ./cmd/migrate up
cd backend && go run ./cmd/api
cd backend && go run ./cmd/worker
cd frontend && npm run typecheck && npm run build
docker compose up -d kafka postgres
```

## Risky files / rollback points

- `backend/internal/service/message_service.go` — Send 主路径；produce 必须 nil-safe（Kafka 缺失不阻塞发消息）。**回滚点**：移除 EventProducer 调用即恢复。
- `backend/internal/hub/hub.go` — presence 回调；nil-safe。**回滚点**：不设 PresencePublisher。
- `backend/cmd/worker` — 新进程；**回滚点**：不启动 worker 即无离线推送，WS 不受影响。
- `frontend/public/sw.js` — SW 有缓存更新问题；dev 用 `navigator.serviceWorker.register` + 版本号，避免旧 SW 残留。

## Notes

- 实现顺序：B1→B2→B3→B4→B5→B6→B7→B8→B9→B10→B11→F1→F2→F3→C1→C2→C3。
- 每次验证通过再进下一步；`go vet` / `tsc` 干净。
- `implement.jsonl` / `check.jsonl` 需在 `task.py start` 前填入真实 spec/research 条目（sub-agent 上下文用）。
