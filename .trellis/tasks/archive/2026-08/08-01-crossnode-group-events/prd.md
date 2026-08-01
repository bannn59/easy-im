# 跨节点组事件广播

## Goal

让 `members.changed` 与 `conversation.renamed` 两类组事件从「本地节点广播」升级为「跨节点广播」，与现有的消息事件（created/edited/recalled/read）走同一条 Kafka fanout 链路。这样在多个 API 节点部署时，用户在节点 A 拉人/改名，节点 B 的在线成员也能实时收到事件，无需轮询或刷新。

## Requirements

- 节点 A 触发组事件（加人/踢人/退群/转让群主/重命名群聊）后，事件写入 Kafka 总线。
- 其他节点（B/C/...）的 fanout consumer 消费该事件并投递到本节点在线的群成员。
- 事件发布必须 **best-effort**：发布失败不能阻塞或失败主操作（沿用现有 `publishRead` / `MessageEventPublisher` 的 nil-safe + 忽略错误模式）。
- worker 离线推送 consumer 不能因新事件类型产生误推送（现有 `EventType() != MessageCreated` 过滤需继续成立）。
- 本地节点不重复投递：origin-skip 已覆盖（与现有 fanout 一致）。

## Acceptance Criteria

- [ ] `im.messages` topic 上新增 `members.changed` 与 `conversation.renamed` 两类事件类型（扩展 `MessageEventType` + `MessageEvent` 字段，`omitempty` 保证旧 consumer 兼容）。
- [ ] `ConversationService` 在广播组事件到本地 hub 前，同时把事件发布到总线（仿 `WithReadPublisher` 模式）。
- [ ] fanout consumer (`FanoutHandler`) 消费这两类事件，重建对应的 WS 帧并投递给本节点在线成员；payload 完整（`members.changed` 含 `action/user_id/members`，`conversation.renamed` 含 `title/updated_at`），无需回查 DB。
- [ ] 新增 fanout 单元测试：members.changed 跨节点投递、conversation.renamed 跨节点投递、own-origin skip、未知类型 skip。
- [ ] 新增 mq topic 测试：两种新事件的 JSON round-trip + 默认类型兼容性不受破坏。
- [ ] 既有测试全绿：`go test ./...`（backend）+ `npm run typecheck`（frontend，若涉及）。
- [ ] 端到端：起两个 API 节点，节点 B 用户 WS 收到节点 A 触发的 `members.changed` / `conversation.renamed`。

## Notes

- 复用现有 `im.messages` topic，**不新增 topic**：fanout consumer 和 worker 已经订阅它，复用成本最低；`MessageEventType` 是现成的判别器。
- `MessageEvent` 是**共享结构体**（fanout consumer + worker + 各 producer adapter 都用它）。新增字段必须 `omitempty`，否则旧 consumer（`ToDomain`）解码旧记录会受影响——虽然本任务不涉及 `ToDomain` 用到新字段，但保持约定。
- `conversation.renamed` 的 payload 需携带 `title` 与 `updated_at`，fanout consumer 才能直接重建 WS 帧而不回查 DB。
- `members.changed` 的 payload 需携带完整的新成员 id 列表（`members`），fanout consumer 直接用它对 `PublishToUsers` 作用域限定。
- 前端 **无需改动**：`conversation.renamed` 与 `members.changed` 的 WS 帧形状不变，前端已能处理。验证型改动即可（可选）。
- 参考实现：`internal/app/fanout.go`（FanoutHandler 模式）、`internal/app/mq_adapter.go`（messageEventAdapter 模式）、`internal/service/conversation_service.go`（broadcastMembersChanged / broadcastRenamed / publishRead）、`internal/mq/topics.go`（MessageEvent 结构）、`cmd/worker/main.go`（worker 过滤）。
