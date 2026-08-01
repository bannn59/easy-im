# 跨节点组事件广播 — 执行计划

## 前置

- [ ] 读 `internal/mq/topics.go`（MessageEvent 结构 + 构造器）确认字段命名风格
- [ ] 读 `internal/app/mq_adapter.go`（messageEventAdapter）确认 adapter 方法签名风格
- [ ] 读 `internal/app/fanout.go`（FanoutHandler switch）确认插入位置

## 执行步骤（按依赖顺序）

### 1. mq 层：事件类型 + 构造器

- [ ] `topics.go`：`MessageEventType` 增加 `GroupMembersChanged` / `GroupConversationRenamed` 常量
- [ ] `MessageEvent` 增加字段：`Action`、`ActorID`、`MemberIDs`、`Title`、`UpdatedAt`（全 `omitempty`）
- [ ] 新增 `NewMembersChangedEvent` / `NewConversationRenamedEvent` 构造器
- **验证**：`go build ./...`

### 2. mq 层：测试

- [ ] `topics_test.go` 新增 `TestGroupEventsRoundTrip`（两种事件 round-trip）
- **验证**：`go test ./internal/mq/`

### 3. app 层：adapter 发布方法

- [ ] `mq_adapter.go`：`messageEventAdapter` 增加 `PublishMembersChanged` / `PublishConversationRenamed`（origin 打标 + TopicMessages + key=conv）
- **验证**：`go build ./...`

### 4. service 层：发布接口 + 接入

- [ ] `conversation_service.go`：新增 `GroupEventPublisher` 接口
- [ ] `ConversationService` 增加 `groupPub` 字段 + `WithGroupEventPublisher`
- [ ] `broadcastMembersChanged` 内追加 `PublishMembersChanged`（nil-safe + slog.Warn）
- [ ] `broadcastRenamed` 内追加 `PublishConversationRenamed`（同上）
- **验证**：`go build ./...`

### 5. app 层：fanout consumer 消费新事件

- [ ] `fanout.go` `FanoutHandler` switch 增加：
  - `mq.GroupMembersChanged` → 重建 `members.changed` 帧，投递 `ev.MemberIDs`
  - `mq.GroupConversationRenamed` → 重建 `conversation.renamed` 帧，投递 `Members.ListMemberIDs`
- **验证**：`go build ./...`

### 6. app 层：fanout 测试

- [ ] `fanout_test.go` 新增 `TestFanoutMembersChangedDelivers` / `TestFanoutConversationRenamedDelivers`
- **验证**：`go test ./internal/app/`

### 7. service 层：发布断言测试（可选但建议）

- [ ] `conversation_service_test.go`：fake publisher 断言 `RenameGroup` 发布 `conversation.renamed` 事件
- **验证**：`go test ./internal/service/`

### 8. 端到端验证

- [ ] 起 2 个 API 节点（`PORT=8081`/`PORT=8082`），同一 DB/Kafka
- [ ] 节点 B 用户 WS 连接，节点 A 触发加人/改名 → 节点 B 收到 `members.changed` / `conversation.renamed`
- **验证**：两个节点的日志 + WS 客户端输出

## 退出标准（对应 PRD AC）

- [ ] 总线事件 round-trip 测试通过
- [ ] fanout 跨节点投递测试通过（members.changed + conversation.renamed）
- [ ] own-origin skip / 未知类型 skip 不回归
- [ ] worker 不因新事件误推送（`EventType() != MessageCreated` 过滤）
- [ ] `go test ./...` 全绿
- [ ] 端到端两节点实测收到事件

## 回滚

- 全部为增量改动（新类型、新字段 omitempty、新 adapter 方法、fanout 新 case、service 新可选发布），无迁移、无破坏性变更。revert 相关 commit 即可。
- 若发布端出现未预期行为，`WithGroupEventPublisher` 不接线即可退回纯本地广播（发布接口 nil-safe）。
