# Implement — 群聊：建群 + 收发消息 + 成员面板

**任务**: `08-01-group-chat`  
**日期**: 2026-08-01

---

## 步骤（ordered checklist）

### Step 1: 后端 `CreateGroup` service

**改动**:
- `internal/service/conversation_service.go`：新增 `CreateGroup(ctx, selfUserID string, title *string, memberIDs []string)`。
- 校验：成员去重 + 剔除 self；总成员 ≥ 2；≤ 50；每个成员 `friends.AreFriends`；`users.FindByID` 存在性。
- 调 `convs.Create`（复用，含 title + 多成员）→ `convs.GetIfMember` 返回。

**验证**:
- `go build ./...`。
- 单测：`conversation_service_test.go` 加 `TestCreateGroup*`（成功/非好友/成员过少/去重）。

### Step 2: 后端 handler + 路由

**改动**:
- `internal/handler/conversation.go`：`CreateGroup` handler（解析 body `{title, member_ids}` → `Conv.CreateGroup` → `201` + DTO）。
- `internal/handler/router.go`：`mux.Handle("POST /v1/conversations/groups", require(...))`。

**验证**:
- `go build ./...`。
- 路由冲突检查：`POST /v1/conversations/groups` 与现有路由不冲突。

### Step 3: 后端测试 + 全量回归

**改动**:
- 补齐 handler 测试（`conversation_test.go` 或 `httpx_test.go` 模式）。
- 手动 curl：建群（好友）、非好友 403、成员 <2 400。

**验证**:
- `go test ./...` 全绿。

### Step 4: 前端创建群对话框

**改动**:
- `src/api/conversations.ts`：`createGroup(title, memberIds)` → `POST /v1/conversations/groups`。
- 新组件 `src/features/chat/CreateGroupDialog.tsx`：多选好友（`listFriends`）+ 群名 + 创建 → `navigate('/c/'+id)`。
- `AppShell` 会话列表工具栏加「创建群」入口。

**验证**:
- `npm run build` 通过。

### Step 5: 前端成员面板

**改动**:
- `ConversationRoom` 标题栏「成员」按钮（仅 `isGroup` 显示）→ 侧栏列 `conv.members`（含 online 状态）。
- 新组件 `src/features/chat/MemberPanel.tsx`（或内联）。

**验证**:
- `npm run build` 通过。
- 手动：打开群会话 → 成员面板显示成员。

### Step 6: i18n 文案

**改动**:
- `src/i18n/index.ts` 中英文：创建群、群名、选成员、创建、成员面板等。

**验证**:
- `npm run build`；切换语言文案正常。

### Step 7: 手动 E2E + 收尾

**改动**:
- 双浏览器：A/B 好友 → A 建群（含 B）→ 双方刷新见群 → 互发消息实时收 → 成员面板正确。

**验证**:
- `go test ./...` + `npm run build` 全绿。
- E2E 通过。
