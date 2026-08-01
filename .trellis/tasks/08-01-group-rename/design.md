# 群名称编辑 — 技术设计

## 1. 数据流

```
前端（成员面板「重命名群聊」）
  → PATCH /v1/conversations/{id}  body: {"title":"新名字"}
  → handler.RenameGroup
      → service.RenameGroup (requireOwner 权限校验)
          → repo.SetTitle (UPDATE conversations SET title, updated_at)
          → hub.PublishToUsers(成员, Event{Type:"conversation.renamed", Payload:{conversation_id,title,updated_at}})
  → 200 {conversation:{...}} (更新后的会话)
WS 广播 → 各成员前端 useRealtime.onConversationRenamed
  → 会话列表 AppShell: 更新该项 title + updated_at + 重新排序
  → 打开的会话 ConversationRoom: 更新 room 标题
```

## 2. 契约

### HTTP

`PATCH /v1/conversations/{id}`

请求体：
```json
{ "title": "新群名" }
```

响应（200）：
```json
{ "conversation": { "id": "...", "title": "新群名", "created_by": "...", "created_at": "...", "updated_at": "...", ... } }
```
与现有 `GET /v1/conversations/{id}` 相同的 `conversationDTO` 形状（`toConversationDTO`）。

错误：
- 400 `invalid`：title 缺失、非字符串、或 trim 后为空
- 403 `forbidden`：非群主（`requireOwner`）
- 404 `not_found`：非成员或会话不存在（`GetIfMember`）

### WS 事件

`conversation.renamed`
```json
{ "conversation_id": "...", "title": "新群名", "updated_at": "2026-08-01T..." }
```
- 发给**所有当前成员**（`ListMemberIDs`），无 except（发起者也收，与 message.read 一致）。
- 仅单机 hub 广播；跨节点事件不在范围（见 PRD Notes）。

## 3. 后端改动

| 文件 | 改动 |
|------|------|
| `internal/repo/conversation_repo.go` | `ConversationStore` 接口 + `SetTitle(ctx, convID, title string) error`，`UPDATE conversations SET title=$2, updated_at=now() WHERE id=$1`（沿用 SetOwner 模式） |
| `internal/service/conversation_service.go` | `RenameGroup(ctx, conversationID, operatorID string, title *string) (domain.Conversation, error)`：`requireOwner` → trim 空白拒绝 → `SetTitle` → `ListMemberIDs` → `broadcastRenamed`；新增 `broadcastRenamed`（仿 `broadcastMembersChanged`） |
| `internal/handler/conversation.go` | `renameGroupBody{Title *string}`；`RenameGroup(w,r,conversationID)`：解码 body → `Conv.RenameGroup` → `writeJSON(200, {conversation: toConversationDTO(...)})` |
| `internal/handler/router.go` | `mux.Handle("PATCH /v1/conversations/{id}", require(...))`（复用现有 `{id}` PathValue 包装，仿 List 的 {id} 处理） |

注意：现有 `Get` handler 从 `r.URL.Path` 手动解析 id（`strings.TrimPrefix`），而 `AddMembers` 等走 `r.PathValue("id")`。新路由用 `PathValue("id")`（路由包装里传参），与成员管理一致。

## 4. 前端改动

| 文件 | 改动 |
|------|------|
| `src/api/conversations.ts` | `renameGroup(id, title): Promise<Conversation>` → `PATCH /v1/conversations/{id}`，body `{title}`，返回 `.conversation` |
| `src/realtime/index.tsx` | `ConversationRenamedData` 类型 `{conversation_id,title,updated_at}`；`RealtimeHandlers.onConversationRenamed?`；dispatch `case 'conversation.renamed'`；useRealtime proxy 转发 |
| `src/features/chat/ConversationRoom.tsx` | 成员面板加「重命名群聊」按钮（仅 `isOwner`），内联输入+保存（仿 AddMembersDialog 的局部表单，或简单 prompt 式——见 Notes）；`onConversationRenamed` 命中本会话时 `setConv` 更新 title |
| `src/app/AppShell.tsx` | `onConversationRenamed`：`setItems` 更新该会话 `title` + `updated_at`，再 `sortConversations`（标题变更不影响排序逻辑，`updated_at` 由后端置 now，仍按 last_message/updated_at 排序） |
| `src/i18n/locales/{zh-CN,en}.json` | `chat.renameGroup`、`chat.renameGroupPlaceholder`、`chat.save`（或复用现有） |

## 5. 测试

### service 层（conversation_service_test.go，用内存 store）
- `TestRenameGroup`：owner 改名成功，store 收到新 title，广播事件含 title
- `TestRenameGroupNonOwnerForbidden`：非 owner → 403
- `TestRenameGroupNonMember`：非成员 → 404
- `TestRenameGroupBlankRejected`：空/全空白 title → 400

需在测试内存 store（`memStore`/类似）上补 `SetTitle` 方法（现有 handler 测试的 `memConvForHandler` 也要补，见下）。

### handler 层（group_test.go）
- `memConvForHandler` 补 `SetTitle`（更新 `items[id].Title` + `CreatedBy` 校验逻辑由 service 负责）
- `TestRenameGroupHandlerSuccess`：200 + title
- `TestRenameGroupHandlerBlank`：400
- `TestRenameGroupHandlerNonOwner`：403

## 6. 明确不做（Out of scope）

- 跨节点（Kafka）重命名事件——journal 已记录 `members.changed` 目前也只有本地广播，本次沿用。
- 群头像编辑、群公告、名称长度硬限制。
- 列表页对「重命名事件」之外的数据刷新逻辑改动。

## 7. Notes / 决策点

- **前端交互形式**：成员面板空间有限。采用「点击重命名 → 标题行变为输入框 + 保存/取消 → 保存调 API」的轻量内联表单；不引入新对话框组件（现有 AddMembersDialog 是独立模态，重命名更轻，内联更贴合）。
- **1:1 会话**：不暴露重命名入口（仅群组 `isGroup` 显示成员面板；成员面板内重命名按钮仅 `isOwner`）。`conversationDTO.title` 对 1:1 恒为 nil，无歧义。
- **失败处理**：前端保存失败沿用 `memberNotice`（现有成员面板错误行）展示。
