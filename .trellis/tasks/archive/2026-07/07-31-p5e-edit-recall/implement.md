# Implement: P5.e Message Edit & Recall

## Implementation Order

后端（migration → domain → repo → service → handler → router），前端（类型 → realtime → bubble UI → 编辑模式 → 会话列表预览）。

---

### Step 1: Migration

**Files:** `backend/migrations/20260731100000_message_edit_recall.sql`

- [x] `+goose Up`: `ALTER TABLE messages ADD COLUMN edited_at TIMESTAMPTZ NULL, ADD COLUMN recalled_at TIMESTAMPTZ NULL`
- [x] `+goose Down`: 两个 DROP COLUMN

**Verify:** `go build ./...`

### Step 2: Domain

**Files:** `backend/internal/domain/message.go`

- [x] `Message` 加 `EditedAt *time.Time` / `RecalledAt *time.Time`

**Verify:** `go build ./...`

### Step 3: Repo

**Files:** `backend/internal/repo/message_repo.go`

- [x] `messageSelectCols` 加 `edited_at, recalled_at`
- [x] `scanMessage` 加两个 scan 目标
- [x] `UpdateBody(ctx, id, body, editedAt) (Message, error)` — RETURNING + head preview 条件更新
- [x] `MarkRecalled(ctx, id, recalledAt) (Message, error)` — RETURNING + head preview "[recalled]" 条件更新

**Verify:** `go build ./...`

### Step 4: Service

**Files:** `backend/internal/service/message_service.go`

- [x] `MessageStore` 接口加 `UpdateBody` / `MarkRecalled`
- [x] `requireOwnRecent(ctx, conversationID, messageID, userID, now)` 私有校验
- [x] `broadcast` 加 eventType 参数（3 个调用点更新）
- [x] `messagePayload` 加 `edited_at` / `recalled_at`
- [x] `Edit(ctx, conversationID, messageID, userID, body) (MessageView, error)`
- [x] `Recall(ctx, conversationID, messageID, userID) (MessageView, error)`

**Verify:** `go build ./...` + 测试更新（memMsg mock 加方法）

### Step 5: Handler & Router

**Files:** `backend/internal/handler/message.go`, `backend/internal/handler/router.go`

- [x] `toMessageDTO` 加 `edited_at` / `recalled_at`（与 WS payload 一致）
- [x] `Edit` handler（PATCH，decode body）
- [x] `Recall` handler（POST）
- [x] router.go 注册两个新路由（require 保护）

**Verify:** `go build ./...`

### Step 6: 后端测试

**Files:** `backend/internal/service/message_service_test.go`

- [x] memMsg 加 UpdateBody/MarkRecalled
- [x] Edit：成功、5 分钟外拒绝、非自己消息拒绝、已撤回拒绝、空/超长 body 拒绝
- [x] Recall：成功、窗口过期拒绝、非自己消息拒绝、重复撤回拒绝
- [x] broadcast 事件类型正确（message.edited / message.recalled）

**Verify:** `go test ./...` 全绿

### Step 7: 前端 — 类型 + Realtime

**Files:** `frontend/src/api/messages.ts`, `frontend/src/realtime/index.ts`, `frontend/src/features/chat/types.ts`

- [x] `Message` 加 `edited_at?` / `recalled_at?`
- [x] `RealtimeHandlers` 加 `onMessageEdited` / `onMessageRecalled`
- [x] dispatch switch 加两个 case
- [x] `useRealtime` proxy 转发两个新回调

**Verify:** `tsc --noEmit`

### Step 8: 前端 — ConversationRoom 处理

**Files:** `frontend/src/features/chat/ConversationRoom.tsx`

- [x] `onMessageEdited` / `onMessageRecalled` → `mergeMessage(prev, toChatItem(m))`
- [x] `api/messages.ts` 加 `editMessage` / `recallMessage` API 函数

**Verify:** `tsc --noEmit`

### Step 9: 前端 — MessageBubble 编辑/撤回 UI

**Files:** `frontend/src/features/chat/MessageBubble.tsx`, `MessageList.tsx`, `ConversationRoom.tsx`

- [x] MessageBubble props 加 `onEdit?` / `onRecall?`
- [x] 自己的消息 + sent + 5 分钟内 → 显示 编辑/撤回 按钮
- [x] `recalled_at` → 气泡体显示占位符（不显示原文）
- [x] `edited_at` → meta 显示 "已编辑"
- [x] 编辑模式：MessageBubble 内 textarea + 保存/取消
- [x] MessageList 透传 onEdit/onRecall
- [x] ConversationRoom 定义 onEdit（保存 → editMessage → mergeMessage）/ onRecall（确认 → recallMessage）

**Verify:** `tsc --noEmit`

### Step 10: 前端 — 会话列表预览 + i18n

**Files:** `frontend/src/app/AppShell.tsx`, `frontend/src/i18n/locales/en.json`, `frontend/src/i18n/locales/zh-CN.json`

- [x] AppShell 的 `onMessageEdited` / `onMessageRecalled` 更新会话 preview
- [x] i18n：`chat.edited`, `chat.recalled`, `chat.recalledPreview` 等 keys（en + zh-CN）

**Verify:** `tsc --noEmit`

---

## Risky Files / Rollback Points

| File | Risk | Rollback |
|------|------|----------|
| `message_repo.go` | scanMessage 加列改错导致 scan 错位 | 编译器保护 |
| `message_service.go` | broadcast 签名变更影响现有调用 | 3 个调用点一次改完 |
| `MessageBubble.tsx` | 编辑模式 UI 复杂度 | 可先只做按钮 + 占位符，编辑模式后置 |

## Validation Commands

```bash
# Backend
go build ./...
go test ./...

# Frontend
npx tsc --noEmit

# Manual
# 1. A 发消息 → 编辑 → B 实时看到新正文
# 2. A 撤回 → 双方看到占位符
# 3. 5 分钟后 → 按钮消失，后端拒绝
```
