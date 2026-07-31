# P5.e Message Edit & Recall

## Goal

Allow a user to edit their own sent messages (within a time window) and recall messages that have already been delivered, with recall showing a "message recalled" placeholder rather than a hard delete.

## Background

P5.e was tagged **Medium risk** in the roadmap (event versioning). The current Message model is append-only with no edit/recall columns, and MessageRepo has no UPDATE methods.

### Current technical foundation (confirmed by research)

- `Message` domain: ID, ConversationID, SenderID, Body, ClientMsgID, Seq, CreatedAt, ReplyToMessageID
- No `edited_at` / `recalled` columns in messages table
- MessageRepo: Insert/List/FindByID/FindByIDs — no Update
- `broadcast` publishes `message.created` to all conversation members including sender
- HTTP DTO and WS payload must match (spec mandate)
- `mergeMessage` upserts by id — edits merge cleanly, recall needs filter/flag
- 15s polling fallback re-fetches full list from `List` — edits/recalls must persist there

## Scope (confirmed)

- **Edit**: own messages only, within 5-minute window after send
- **Recall**: own messages only, within 5-minute window; shows "recalled" placeholder (Option B), not hard delete
- Conversation list preview updates on edit (new body) and recall ("[recalled]")

## Requirements

### R1 — 编辑消息

- 用户可编辑自己的消息，且仅在发送后 5 分钟内
- 编辑后广播 `message.edited` 事件（含完整 message DTO + `edited_at`）
- 消息列表、会话列表预览显示新正文

### R2 — 撤回消息

- 用户可撤回自己的消息，且仅在发送后 5 分钟内
- 撤回后广播 `message.recalled` 事件
- 所有客户端显示 "对方撤回了一条消息" 占位符（原文不再可见）
- 会话列表预览显示 "[已撤回]"

### R3 — 数据模型

- messages 表新增 `edited_at TIMESTAMPTZ NULL` 和 `recalled_at TIMESTAMPTZ NULL`
- Message domain 新增 `EditedAt` / `RecalledAt`（指针）
- List 返回完整消息（含 recalled 标记），由前端渲染占位符

### R4 — API & 实时

- `PATCH /v1/conversations/{id}/messages/{messageID}` — 编辑
- `POST /v1/conversations/{id}/messages/{messageID}/recall` — 撤回
- WS 事件：`message.edited` / `message.recalled`（payload = 完整 message DTO）
- HTTP DTO 和 WS payload 形状一致

### R5 — 前端

- MessageBubble 气泡菜单：编辑 / 撤回按钮（仅自己的消息 + 5 分钟内）
- 编辑：气泡进入编辑模式，保存后更新正文
- 撤回：气泡显示 "已撤回" 占位符
- 5 分钟窗口：前端禁用按钮 + 后端强制校验（后端为准）

## Acceptance Criteria

- [ ] 用户 A 编辑自己 5 分钟内发送的消息 → 用户 B 实时看到更新后的正文
- [ ] A 尝试编辑 5 分钟前的消息 → 被拒绝（后端返回错误）
- [ ] A 不能编辑 B 的消息（后端返回 forbidden/not_found）
- [ ] A 撤回消息 → 双方都看到 "对方撤回了一条消息" 占位符，原文不可见
- [ ] 撤回后刷新页面，消息仍显示占位符（List 持久化）
- [ ] 会话列表预览：编辑后显示新正文，撤回后显示 "[已撤回]"
- [ ] 编辑/撤回 5 分钟窗口过期后按钮消失/禁用
- [ ] `message.edited` / `message.recalled` WS 事件正确处理（含 unknown 事件不崩溃）
- [ ] HTTP DTO 与 WS payload 字段一致

## Out of Scope

- 编辑历史 / 版本回滚（只保留最新编辑）
- 管理员强制撤回
- 超过 5 分钟的撤回（微信也是限时）
- 图片/文件的编辑（当前仅文本消息）
- 已读回执与撤回的联动（撤回后已读状态如何处理）

## Open Questions

_(none — all blocking decisions resolved)_
