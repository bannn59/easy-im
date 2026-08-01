# 跨节点组事件广播 — 技术设计

## 1. 数据流

```
节点 A                          Kafka                        节点 B
ConversationService            im.messages (key=conv)        fanout consumer (group easyim-realtime-<nodeB>)
  member op / rename ──► publish GroupEvent ─────► ──► FanoutHandler
  └─► hub.PublishToUsers (本地)                       │
                                                     │ origin == nodeB? skip
                                                     │ origin == nodeA (≠ nodeB)
                                                     └─► 重建 WS 帧 → hub.PublishToUsers(本节点在线成员)
worker (group easyim-worker-offline-push)
  └─► EventType() != created → skip (不误推送)
```

复用现有 `im.messages` topic + `MessageEvent` 结构体 + fanout consumer。不新增 topic、不新增 consumer group。

## 2. 契约

### 总线事件（Kafka `im.messages`，`MessageEventType` 扩展）

新增两种 `MessageEventType`：

| Type | 场景 | 关键字段（新增，均 `omitempty`） |
|------|------|------|
| `group.members_changed` | 加人/踢人/退群/转让群主 | `Action`, `ActorID`, `MemberIDs` |
| `group.conversation_renamed` | 重命名群聊 | `Title`, `UpdatedAt` |

沿用现有 `MessageEvent` 公共字段：`Type`、`Origin`、`ConversationID`、`ID`(空)。

> 命名理由：`members.changed` 是 WS 帧名，为避免和 WS 帧字符串混淆，总线类型用 `group.members_changed` / `group.conversation_renamed`。二者在 `FanoutHandler` 内映射回 WS 帧名 `members.changed` / `conversation.renamed`。

### WS 帧（不变，前端已支持）

- `members.changed`：`{conversation_id, action, user_id, members}`（`action`: `added|left|kicked|owner_transferred`）
- `conversation.renamed`：`{conversation_id, title, updated_at}`

fanout consumer 从总线事件直接重建这两个帧，**不回查 DB**。

## 3. 后端改动

### 3.1 `internal/mq/topics.go` — 事件类型 + 构造器

- `MessageEventType` 增加 `GroupMembersChanged` / `GroupConversationRenamed`。
- `MessageEvent` 增加字段（全部 `omitempty`）：
  - `Action string json:"action,omitempty"`
  - `ActorID string json:"actor_id,omitempty"`
  - `MemberIDs []string json:"member_ids,omitempty"`
  - `Title string json:"title,omitempty"`
  - `UpdatedAt time.Time json:"updated_at,omitempty"`
- 新增构造器：
  - `NewMembersChangedEvent(conversationID, action, actorID string, members []string, origin string) MessageEvent`
  - `NewConversationRenamedEvent(conversationID, title string, updatedAt time.Time, origin string) MessageEvent`

### 3.2 `internal/app/mq_adapter.go` — 发布 adapter

`messageEventAdapter` 增加两个方法：
- `PublishMembersChanged(ctx, conversationID, action, actorID string, members []string) error`
- `PublishConversationRenamed(ctx, conversationID, title string, updatedAt time.Time) error`

每个方法内建 `Origin: a.nodeID`，发布到 `TopicMessages`，key=`conversationID`。

### 3.3 `internal/service/conversation_service.go` — 发布接口 + 接入

- 新增窄接口（仿 `ReadEventPublisher`）：
  ```go
  type GroupEventPublisher interface {
      PublishMembersChanged(ctx context.Context, conversationID, action, actorID string, members []string) error
      PublishConversationRenamed(ctx context.Context, conversationID, title string, updatedAt time.Time) error
  }
  ```
- `ConversationService` 增加 `groupPub GroupEventPublisher` 字段 + `WithGroupEventPublisher(p)`。
- 在 `broadcastMembersChanged` 与 `broadcastRenamed` 内，本地 hub 广播之外**追加总线发布**（`s.groupPub != nil` 时调用，失败仅 `slog.Warn`，不阻塞）：
  - `broadcastMembersChanged` → `PublishMembersChanged`
  - `broadcastRenamed` → `PublishConversationRenamed`

### 3.4 `internal/app/fanout.go` — consumer 消费新事件

`FanoutHandler` 的 switch 增加两个 case：
- `mq.GroupMembersChanged`：用 `ev.Action/ActorID/MemberIDs` 重建 `members.changed` 帧，投递到 `ev.MemberIDs`（事件自带成员列表，不查 DB）。
- `mq.GroupConversationRenamed`：用 `ev.Title/UpdatedAt` 重建 `conversation.renamed` 帧，投递前需 `Members.ListMemberIDs` 确定当前成员（因为改名事件不携带成员列表）。

> 注意区分：`members.changed` 事件携带 `MemberIDs`（直接作用域）；`conversation.renamed` 事件不携带成员列表，用 `Members.ListMemberIDs` 现查（与消息事件一致）。

## 4. worker 过滤验证

`cmd/worker/main.go` 已有 `if ev.EventType() != mq.MessageCreated { return nil }`。新增的两种类型不属于 `MessageCreated`，天然被过滤，不会触发离线推送。**无需改动**，只需测试确认。

## 5. 测试

### mq 层（`topics_test.go`）
- `TestGroupEventsRoundTrip`：两种新事件 marshal/unmarshal round-trip，字段完整。
- 现有 `TestMessageEventDefaultTypeIsCreated` 已保证旧记录兼容，无需改动。

### app 层（`fanout_test.go`）
- `TestFanoutMembersChangedDelivers`：非 own-origin 的 `group.members_changed` → 投递 `members.changed` 帧到 `MemberIDs`。
- `TestFanoutConversationRenamedDelivers`：非 own-origin 的 `group.conversation_renamed` → 投递 `conversation.renamed` 帧到 `Members.ListMemberIDs`。
- 现有 `TestFanoutSkipsOwnOrigin` 覆盖 own-origin skip（复用）。

### service 层（可选，补一个）
- `TestRenameGroupPublishesEvent` / `TestMembersChangedPublishesEvent`：用 fake publisher 断言事件发布（验证 `groupPub` 接线）。用现有 `membersTestHarness` 改造成带 publisher 的版本。

## 6. 明确不做（Out of scope）

- 前端改动：WS 帧不变，前端已支持；不需要改。
- 独立新 topic：不新增，复用 `im.messages`。
- 事件重放/补偿、Kafka 事务：维持现有 best-effort 语义。
- 组事件的历史持久化（`conversation_events` 表）：不在范围。

## 7. 风险与决策点

- **事件命名**：总线 `group.members_changed` vs WS `members.changed` 两套字符串。fanout 内映射一次，避免歧义。备选方案是直接复用 WS 帧名作为总线类型（少一次映射但语义混叠），我选择前者（类型名表达总线语义）。
- **`conversation.renamed` 不带成员列表**：改名不改变成员，重建帧时用 `Members.ListMemberIDs` 现查，与消息事件一致。若未来成员列表变化频繁可优化，但当前正确。
- **消息丢失窗口**：best-effort 发布，总线故障时跨节点事件丢失（本地广播仍工作）。这与现有消息事件语义一致，接受。
- **顺序**：`im.messages` 按 `conversation_id` 分区，组事件与消息事件同 key，同会话内顺序有保证。
