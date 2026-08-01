# Implement — 群成员管理：拉人 / 退群 / 踢人 / 转让群主

**任务**: `08-01-group-members`  
**日期**: 2026-08-01

---

## 步骤（ordered checklist）

### Step 1: repo 成员增删 + 群主更新

**改动**:
- `internal/repo/conversation_repo.go`：新增 `AddMembers` / `RemoveMember` / `SetOwner`。
- `internal/service/conversation_service.go`：`ConversationStore` 接口加 3 个方法。

**验证**:
- `go build ./...`。
- 现有 memConv mock 需补 3 方法（`conversation_service_test.go` + `group_test.go` 的 mock）。

### Step 2: service 4 个方法 + 权限校验

**改动**:
- `AddMembers`（成员校验 + 好友 + 不在群）
- `LeaveGroup`（非群主可退，最后成员/群主禁止）
- `KickMember`（群主专属）
- `TransferOwner`（群主转让）

**验证**:
- `go build ./...`。
- service 单测：权限矩阵（见 design §8）。

### Step 3: handler + 路由

**改动**:
- `internal/handler/conversation.go`：`AddMembers` / `LeaveGroup` / `KickMember` / `TransferOwner` handler。
- `internal/handler/router.go`：4 个路由（检查 `me` 精确段与 `{userID}` 通配段不冲突）。

**验证**:
- `go build ./...`。
- handler 单测：状态码 + 路由映射。

### Step 4: WS `members.changed` 广播

**改动**:
- service 各方法完成后广播 `members.changed`（action + user_id + members 完整列表）。
- 新增 `broadcastMembersChanged(ctx, conversationID, action, userID string)` 辅助，拉完整成员列表。

**验证**:
- `go build ./...`。
- 手动：开 2 个 WS，A 拉 B → B 收到 `members.changed`。

### Step 5: 前端 API + 实时事件

**改动**:
- `src/api/conversations.ts`：`addMembers` / `leaveGroup` / `kickMember` / `transferOwner`。
- `src/realtime/index.tsx`：`RealtimeHandlers` 加 `onMembersChanged` + dispatch。

**验证**:
- `npm run build`。

### Step 6: 前端成员面板操作

**改动**:
- `ConversationRoom` 成员面板：拉人入口（抽 MemberPicker 复用多选）、退出群聊、群主视图踢人/转让。
- `AppShell`/`ConversationRoom` 消费 `onMembersChanged`：更新 `conv.members` + 刷新会话列表。

**验证**:
- `npm run build`。
- 手动 E2E：3 人群 → 拉人/退群/踢人/转让全流程。

### Step 7: 全量回归 + 收尾

**改动**:
- `go test ./...` + `npm run build` 全绿。
- spec 更新（如需：quality-guidelines ACL 约定补成员事件）。
