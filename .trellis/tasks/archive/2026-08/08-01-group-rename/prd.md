# 群名称编辑

## Goal

允许群主在聊天界面内重命名一个群聊，名称变更对所有成员实时生效（打开会话的成员即时看到新标题，会话列表预览同步更新）。沿用现有 CreateGroup 的名称规则：名称可选，未命名群显示「群聊 / Group chat」。

## Requirements

- 仅群主（`conversations.created_by`）可重命名群聊。
- 重命名成功后，所有群成员的会话标题实时更新（通过 WS 事件），不依赖刷新。
- 群名规则与建群一致：可选、去除首尾空白；空白/超长名称应被拒绝（沿用 CreateGroup 的宽松处理，具体限制见 Notes）。
- 非群主调接口返回 403；非成员调接口返回 404（与现有群操作一致）。

## Acceptance Criteria

- [ ] 群主在成员面板中看到「重命名群聊」入口；点击后编辑群名并保存。
- [ ] `PATCH /v1/conversations/{id}` 仅群主可改标题，成功返回 200 + 更新后的会话。
- [ ] 空白标题被拒绝（400）；群名修改后 `updated_at` 更新。
- [ ] 重命名后，通过 WS 向所有群成员广播 `conversation.renamed`；在线成员会话标题即时更新，会话列表预览同步。
- [ ] 后端单元测试：service 层（非成员 404 / 非群主 403 / 成功改名）与 handler 层（成功 200 / 空白 400 / 非群主 403）。
- [ ] 前端 i18n：中英双语新增重命名相关文案。
- [ ] `npm run typecheck` 与 `go test ./...` 通过。

## Notes

- 命名沿用 CreateGroup：`title *string`，nil = 未命名群。重命名传空字符串/全空白应返回 400（与建群“空标题”行为对齐——建群时空标题存为 nil，重命名时显式拒绝空白更稳妥）。
- 标题长度：建群时无强制上限（DB `TEXT`），本任务不新增硬限制，只做空白拒绝，保持与建群一致；前端沿用 CreateGroupDialog 的行为（无 maxLength 约束）。
- 无需数据库迁移：`conversations.title` 列已存在。
- 实时事件沿用现有 hub.PublishToUsers + JSON payload 模式（类似 members.changed）；本任务只做单机 WS 广播，跨节点事件（Kafka）明确不在范围。
- 参考现有实现：`CreateGroup`（service/conversation_service.go:127）、`KickMember`/`TransferOwner`（权限模式）、`members.changed`（实时广播模式）、成员面板（ConversationRoom.tsx:456）。
