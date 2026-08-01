# 群名称编辑 — 执行计划

## 前置

- [ ] 读 `backend/internal/repo/conversation_repo.go`（SetOwner 附近）与 `internal/service/conversation_service_test.go`（TestTransferOwner 附近），确认 store 接口与内存 store 结构
- [ ] 确认前端 i18n `chat.*` 现有键，避免重复（已有 `chat.renameGroup` 不存在，需新增）

## 执行步骤（后端先行，验证后做前端）

### 1. 后端 repo：`SetTitle`

- [ ] `ConversationStore` 接口加 `SetTitle(ctx, conversationID, title string) error`
- [ ] `conversation_repo.go` 实现 `SetTitle`：`UPDATE conversations SET title=$2, updated_at=now() WHERE id=$1`
- **验证**：`go build ./...`

### 2. 后端 service：`RenameGroup`

- [ ] `RenameGroup(ctx, conversationID, operatorID string, title *string) (domain.Conversation, error)`
  - `requireOwner`（复用现有方法）→ 非成员 404 / 非群主 403
  - `title == nil || strings.TrimSpace(*title) == ""` → `apperr.Invalid("title is required")`
  - `s.convs.SetTitle(...)` → 成功
  - `ListMemberIDs` → `broadcastRenamed`（新增，payload: `{conversation_id,title,updated_at}`）
  - 返回 `requireMember` 重新读取的会话（或 GetIfMember）
- [ ] 新增 `broadcastRenamed`（仿 `broadcastMembersChanged`，`hub.Event{Type:"conversation.renamed",...}`）
- **验证**：`go build ./...`

### 3. 后端 handler + 路由

- [ ] `conversation.go`：`renameGroupBody{Title *string}`；`RenameGroup(w, r, conversationID)` → 解码 → `Conv.RenameGroup` → `writeJSON(200, {"conversation": toConversationDTO(c, h.Hub)})`
- [ ] `router.go`：`mux.Handle("PATCH /v1/conversations/{id}", require(...))` 传 `r.PathValue("id")`
- **验证**：`go build ./...`

### 4. 后端测试

- [ ] `conversation_service_test.go`：内存 store 补 `SetTitle`；新增 4 个 service 测试（成功/非群主 403/非成员 404/空白 400）
- [ ] `group_test.go`：`memConvForHandler` 补 `SetTitle`；新增 3 个 handler 测试（成功 200/空白 400/非群主 403）
- **验证**：`go test ./...`（backend 全绿）

### 5. 前端 API + realtime

- [ ] `api/conversations.ts`：`renameGroup(id, title)` → `PATCH`，返回 `.conversation`
- [ ] `realtime/index.tsx`：`ConversationRenamedData` 类型 + `onConversationRenamed` handler + dispatch case + useRealtime proxy
- **验证**：`npm run typecheck`

### 6. 前端 UI

- [ ] `ConversationRoom.tsx`：
  - 成员面板加「重命名群聊」按钮（仅 `isOwner`），点击后内联输入框 + 保存/取消
  - 保存调 `renameGroup`，成功更新 `conv`；失败显示 `memberNotice`
  - `onConversationRenamed`：命中本会话 → `setConv` 更新 title
- [ ] `AppShell.tsx`：`onConversationRenamed` 更新列表项 title/updated_at 并重排
- [ ] i18n 两语言新增 `chat.renameGroup`、`chat.renameGroupPlaceholder`、`chat.cancel`（复用 `chat.cancelReply`? 检查后定）
- **验证**：`npm run typecheck`

### 7. 端到端验证

- [ ] `go test ./...`（backend）
- [ ] `npm run typecheck`（frontend）
- [ ] 手动冒烟：起后端 + 前端，群主重命名 → 成员端标题实时更新（若环境允许）

## 退出标准（对应 PRD AC）

- [ ] 群主能重命名，非群主 403，空白 400
- [ ] WS `conversation.renamed` 实时更新会话标题 + 列表预览
- [ ] 全部单测 + typecheck 通过

## 回滚

- 后端纯增量（新接口/新方法/新路由），无迁移、无破坏性改动 → 直接 revert 相关 commit
- 前端同为新 handler + 新 API 函数，无破坏
