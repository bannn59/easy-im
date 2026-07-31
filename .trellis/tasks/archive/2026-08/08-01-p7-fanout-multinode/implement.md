# Implement — 多节点实时扇出（Kafka-backed fanout）

**任务**: `08-01-p7-fanout-multinode`  
**日期**: 2026-08-01

---

## 步骤（ordered checklist）

### Step 1: `internal/mq` — FanoutEvent + producer 扩展

**改动**:
- `topics.go`：新增 `FanoutEvent` 结构（见 design §4.1），`Type` 字段：`created` / `edited` / `recalled` / `read`；`Origin` 字段。
- 保留 `MessageEvent` 兼容：`FanoutEvent` 复用 `MessageEvent` 的字段（ID/ConversationID/SenderID/Body/CreatedAt），新增 `omitempty` 字段（EditedAt/RecalledAt/ReadByUserID/ReadSeq/Type/Origin）。
- `NewFanoutEvent` 构造器。
- `ToDomain()` 兼容 created。

**验证**:
- `go build ./...`
- 单测：worker 用旧 `MessageEvent` 解码含新字段的 `created` 事件成功。

### Step 2: `internal/app` — nodeID + origin 注入

**改动**:
- `api.go`：生成 `nodeID`（`hostname:pid` 或 UUID），传入 producer 适配器。
- `mq_adapter.go`：`PublishMessageCreated` → 发布 `FanoutEvent{Type: "created", Origin: nodeID, ...}`。

**验证**:
- `go build ./...`；现有 message send 测试通过（producer 为 NoopProducer 时不受影响）。

### Step 3: `internal/app` — fanout consumer

**改动**:
- `api.go`：Kafka 配置时，创建 `mq.NewConsumer(Group: "easyim-realtime", ...)`。
- 启动 goroutine 跑 `fanoutConsumer.Run`：
  - 解码 `FanoutEvent`；
  - `if ev.Origin == nodeID { skip }`；
  - `memberIDs := members.ListMemberIDs(ctx, ev.ConversationID)`；
  - 构造 WS payload（复用 message shape）→ `hub.PublishToUsers(memberIDs, ...)`。
- 优雅停机：consumer 随 api 进程退出关闭。

**验证**:
- `go build ./...`；无 Kafka 时不启动 consumer（日志确认）。

### Step 4: `internal/service` — Edit/Recall/Read 发布总线事件

**改动**:
- `MessageService`：`Edit` / `Recall` 后调用 `publishEvent`（新增 `MessageEventPublisher` 方法或泛化）发布 `edited` / `recalled` 事件（带 origin）。
- `ConversationService`：mark read 后发布 `read` 事件。

**验证**:
- `go build ./...`；`go test ./internal/service/` 全绿。
- 新增单测：Edit/Recall 后 publisher 被调用，事件 type 正确。

### Step 5: 双节点 E2E 验证

**脚本思路**（可用 bash + curl + websocket 客户端）:
1. `docker compose up -d`。
2. 启动两个 api 实例（`PORT=8081` / `PORT=8082`，共享 DB/Kafka）。
3. 注册两用户，互加好友，开会话。
4. A 连 node1 的 `/v1/ws`，B 连 node2 的 `/v1/ws`。
5. A 发消息 → 断言 B 的 WS 收到 `message.created`（跨节点）。
6. 断言 A 只收到 1 次（去重）。
7. 停 Kafka 或单节点，验证降级实时。

**验证**:
- 手动脚本通过，输出断言结果。

### Step 6: 回归 + 测试

**改动**:
- `go test ./...` 全绿。
- `cd frontend && npm run build` 全绿（若事件形状变化则改前端）。
- 现有 `07-31-p6-offline-push` 相关测试不回归。

### Step 7: spec 更新

**改动**:
- `.trellis/spec/backend/realtime-messaging.md`：记录多节点 fanout 架构（fanout consumer、origin 去重、事件类型表、topic 策略）。
- `.trellis/spec/backend/index.md` 如有需要补充。

### Step 8: 复盘 + 归档

**改动**:
- journal 记录。
- `task.py` 归档任务（父任务归档、子任务视完成情况）。

---

## 验证命令汇总

```bash
cd backend && go build ./... && go test ./...
cd frontend && npm run build
# 双节点脚本见 Step 5
```

---

## Rollback

- 若 fanout consumer 引入回归：可通过不配置 `KAFKA_BROKERS`（或加 env 开关 `REALTIME_FANOUT=0`）回退到纯本地广播。设计上 consumer 与 producer 同条件启停，回退路径即现状。
- 事件结构向后兼容（`MessageEvent` 字段保留），worker 不受影响。
