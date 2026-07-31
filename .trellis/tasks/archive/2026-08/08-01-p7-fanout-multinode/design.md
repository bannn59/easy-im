# Design — 多节点实时扇出（Kafka-backed fanout）

**任务**: `08-01-p7-fanout-multinode`  
**日期**: 2026-08-01

---

## 1. 目标形态

```
User A (node1)              Kafka im.messages              User B (node2)
   │ WS conn                     │  topic                    │ WS conn
   ▼                             ▼                           ▼
node1 api ──send HTTP──► DB ──produce──► im.messages ──► worker(offline push, group)
   │                                                                    
   │ local hub ──► A (本节点直推)                          
   └── fanout consumer (group easyim-realtime) ◄─────────────► node1 hub
                                                              node2 hub ──► B
```

**核心思路**: 每个 api 节点运行**自己的** fanout consumer（独立消费组），消费 `im.messages`，把事件推给**本节点**连接的收件人。这样任何节点产生的事件，最终会被每个节点各自送达自己的在线用户。

## 2. 双写与去重

### 现状

`MessageService.Send` 已做双路：
1. `broadcast()` — hub 内存直推（本节点在线用户）
2. `publishEvent()` — 发布到 Kafka（`im.messages`），供 worker 离线推送

### 加入 fanout consumer 后的事件流

**发送方节点**：`Send` 落库 → 本地直推（本节点用户）→ 发布 Kafka。Kafka 事件回到自己节点的 fanout consumer → **再次推送给本节点用户**。

→ 发送方节点上的用户会收到 **2 次** `message.created`（1 次本地直推 + 1 次 fanout consumer 回推）。

### 去重方案：来源标记（origin） + 发送节点跳过

**推荐方案**：在事件里带 `origin`（产生该事件的节点/进程 ID）。fanout consumer 收到事件时，**跳过 origin == 自身** 的事件。

- 发送节点：本地直推已覆盖本节点用户 → fanout consumer 丢弃自己的事件 → **本节点用户只收 1 次**。
- 其他节点：origin != 自身 → 推给本节点用户 → **跨节点用户收 1 次**。

**代价**：需要 `origin` 字段 + 每节点唯一 ID（如 `os.Hostname() + pid` 或启动时生成的 UUID）。

**为什么不靠 client_msg_id 去重**：`client_msg_id` 是客户端生成、用于幂等的；用它去重需要维护「每个用户已见 msg id」的内存/DB 集合，复杂且仍可能重复（消费偏移、重启）。origin 跳过更简单、语义更清楚。

**备选（不采用）**：发送节点不本地直推、全走 Kafka 回推。缺点：单节点无 Kafka 时无法实时（违反「兼容单节点」约束）；且所有消息多一跳延迟。

## 3. 事件类型覆盖

当前总线只有 `MessageEvent`（created）。多节点下这些事件也需要跨节点：

| 事件 | 现状 | 跨节点方案 |
|------|------|-----------|
| `message.created` | 走 Kafka（离线） | 复用 `im.messages`，fanout consumer 推 |
| `message.edited` | 仅本地 hub | 需新增总线事件 / 复用同一 topic 带 type |
| `message.recalled` | 仅本地 hub | 同上 |
| `message.read` | 仅本地 hub（conversation_service） | 需新增 |
| `typing.started/stopped` | 仅本地 hub | 需新增 |
| `presence.changed` | 仅本地 hub（friend 广播） | 已有 `im.presence`，但语义不同 |

### 范围决策（MVP）

**PRD 验收 4 说「至少核心消息事件，其余按 design 取舍」。** 建议：

- **本次做**：`message.created` / `message.edited` / `message.recalled` — 跨节点消息一致性最要紧。
- **本次做**：`message.read` — 已读回执跨节点（简单，复用同一 topic + type）。
- **本次可选/后置**：`typing.*` — 高频、短时效、非一致性强；本地 hub 已覆盖单节点。多节点下「对方看不到你打字」可接受（微信也是尽力而为）。**列入 backlog，本次不做**。
- **本次不做**：`presence.changed` 跨节点 — 这是「多节点在线聚合」问题，语义复杂（一个节点 online、另一节点 offline 如何合并），且当前前端 presence 已通过 `IsOnline`/`OnlineUserIDs` 查单节点。**列为后续工作**。

### Topic 策略

- 方案 A：单 topic `im.messages` 加 `type` 字段（`created`/`edited`/`recalled`/`read`）。
  - 优点：分区键 `conversation_id` 保证同一会话内事件有序；一个 topic 好管理。
  - 缺点：worker 需按 type 过滤（当前 worker 只处理 created）。
- 方案 B：每事件类型独立 topic。
  - 优点：worker 干净。
  - 缺点：多个 topic，分区键含义不同，管理复杂。

**选 A**：`im.messages` 加 `type`，key 仍 `conversation_id`。worker 消费时只处理 `type == "created"`（现有 `MessageEvent` 语义默认 created，向后兼容）。

## 4. 组件设计

### 4.1 `internal/mq` 扩展

```go
// FanoutEvent wraps message-related events for cross-node fanout.
type FanoutEvent struct {
    Type           string    `json:"type"`            // "created"|"edited"|"recalled"|"read"
    Origin         string    `json:"origin"`          // node/process id that produced this
    MessageID      string    `json:"message_id"`
    ConversationID string    `json:"conversation_id"`
    // per-type fields...
    Body           string    `json:"body,omitempty"`
    SenderID       string    `json:"sender_id,omitempty"`
    EditedAt       time.Time `json:"edited_at,omitempty"`
    RecalledAt     time.Time `json:"recalled_at,omitempty"`
    ReadByUserID   string    `json:"read_by_user_id,omitempty"`
    ReadSeq        int64     `json:"read_seq,omitempty"`
    CreatedAt      time.Time `json:"created_at"`
}
```

设计要点：
- 保留 `MessageEvent`（离线推送用，向后兼容），或让 `MessageEvent` 成为 `FanoutEvent` 的子集。
- **向后兼容**：worker 已按 `mq.MessageEvent` 解码 `im.messages`。若改字段名/结构，必须保证 worker 仍能解码 `created` 事件。→ **优先：`FanoutEvent` 内嵌或复用 `MessageEvent` 字段**，新字段用 `omitempty`，worker 解码不受影响。

### 4.2 `internal/app` — fanout consumer

```go
// In NewAPIHandler, when Kafka configured:
fanoutConsumer, _ := mq.NewConsumer(mq.ConsumerOpts{
    Brokers:  opts.KafkaBrokers,
    Group:    "easyim-realtime",        // 独立消费组
    ClientID: "easyim-realtime-" + nodeID,
    Topics:   []string{mq.TopicMessages},
    Log:      log,
})
go fanoutConsumer.Run(ctx, func(ctx, msg) error {
    ev := decode FanoutEvent
    if ev.Origin == nodeID { return nil }          // 跳过自身
    // 按 conversation 查本节点成员 → 推给本节点在线成员
    memberIDs := members.ListMemberIDs(ctx, ev.ConversationID)
    payload := buildWSMessage(ev)
    hub.PublishToUsers(memberIDs, hub.Event{Type: wsEventType(ev.Type), Payload: payload})
})
```

- consumer 生命周期挂在 api 进程；优雅停机时关闭。
- `nodeID` 每进程生成（如 `hostname:pid`），同时作为 origin 发布方标记。

### 4.3 `internal/service` 改动

- `MessageService.Send`：发布 Kafka 事件时带上 `origin`（通过 adapter）。
- `Edit` / `Recall`：**新增** 发布 `edited` / `recalled` 事件到 `im.messages`（当前只有本地广播）。
- `ConversationService`（mark read）：**新增** 发布 `read` 事件。
- 消息 payload 复用：`messagePayload(view)` 已是 WS 与 HTTP 共享形状，fanout consumer 可复用同一构造逻辑，保证事件形状一致。

### 4.4 本地直推保留

`broadcast()` 保留——单节点时是唯一路径，多节点时覆盖本节点用户（配合 origin 跳过避免重复）。

## 5. 单节点降级

`KAFKA_BROKERS` 未配置（或 producer 初始化失败）时：

- producer = `NoopProducer`（现有行为）→ 不发布事件。
- fanout consumer 不启动（无 broker）→ 无回推。
- 本地 `broadcast()` 是唯一实时路径 → 功能不退化。

**要点**：fanout consumer 的启动条件与 producer 一致，都必须有 Kafka；无 Kafka 时整套「总线」静默关闭。

## 6. 边界与风险

| 风险 | 影响 | 缓解 |
|------|------|------|
| origin 未实现 → 双写重复 | 同节点用户收 2 次 | origin 跳过是去重的核心，必须实现并测试 |
| worker 解码兼容 | 离线推送坏掉 | 保留 `MessageEvent` 字段；`type` 默认 `created`；单测覆盖 worker 解码 |
| 消费延迟 | 跨节点消息延迟 | Kafka 单 broker 延迟低（ms 级）；可接受 |
| consumer 崩溃 | 跨节点事件丢失 | at-least-once（现有 Consumer.CommitRecords）；重启后从 offset 续读 |
| memberIDs 查询压力 | 每个事件一次 DB 查询 | 高频时优化（如 per-conversation 成员缓存），MVP 先直接查 |
| 每个节点消费全量 im.messages | 节点越多重复消费越多 | 这是 fanout 模型固有成本（每个节点都要知道所有会话消息）；大数据量时可做会话级 sharding，MVP 不做 |

## 7. 不做的事（明确）

- Redis 做在线集合、member cache — 不引入新依赖。
- typing / presence 跨节点 — 列 backlog。
- WebSocket 网关独立进程 — 这是后续 cmd/gateway 工作。

## 8. 验收方法

双实例验证：

1. `docker compose up -d`（Postgres + Kafka）。
2. 启动两个 api 实例（不同 `PORT`），共享同一 Postgres/Kafka。
3. 两个浏览器（或脚本）分别登录不同用户，各自连**不同** api 的 `/v1/ws`。
4. A 发消息 → B 应实时收到（跨节点）。
5. A 发消息 → A 应只收到 1 次（去重）。
6. 停掉一个 Kafka 消费，验证单节点降级仍实时。
