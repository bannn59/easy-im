# Design: P5.e Message Edit & Recall

## Architecture Overview

```
[PATCH /v1/conversations/{id}/messages/{mid}] → MessageService.Edit → MessageRepo.UpdateBody
  → 校验：成员 + 自己的消息 + 5分钟窗口
  → 更新 body + edited_at → 广播 message.edited（完整 DTO）→ 更新会话列表预览

[POST /v1/conversations/{id}/messages/{mid}/recall] → MessageService.Recall → MessageRepo.MarkRecalled
  → 校验同上 → 更新 recalled_at → 广播 message.recalled → 更新会话列表预览 "[已撤回]"
```

- **不硬删除**：recalled 行保留，前端渲染占位符
- **编辑不保留历史**：只覆盖 body + 设 edited_at
- **5 分钟窗口**：后端强制校验（前端仅 UI 提示）

## 1. Migration

**Files:** `backend/migrations/20260731100000_message_edit_recall.sql`

```sql
-- +goose Up
ALTER TABLE messages
  ADD COLUMN edited_at  TIMESTAMPTZ NULL,
  ADD COLUMN recalled_at TIMESTAMPTZ NULL;

-- +goose Down
ALTER TABLE messages
  DROP COLUMN edited_at,
  DROP COLUMN recalled_at;
```

## 2. Domain

**Files:** `backend/internal/domain/message.go`

```go
type Message struct {
    ID               string
    ConversationID   string
    SenderID         string
    Body             string
    ClientMsgID      string
    Seq              int64
    CreatedAt        time.Time
    ReplyToMessageID *string
    EditedAt         *time.Time   // NEW
    RecalledAt       *time.Time   // NEW
}
```

## 3. Repo

**Files:** `backend/internal/repo/message_repo.go`

- `messageSelectCols` 加 `edited_at, recalled_at`；`scanMessage` 加两个 scan 目标
- 新增 `UpdateBody(ctx, id, body string, editedAt time.Time) (domain.Message, error)`：
  - `UPDATE messages SET body=$2, edited_at=$3 WHERE id=$1 RETURNING ...`
  - 注意：编辑后需要更新会话列表 `last_message_preview`（仅当编辑的是最新消息 head）
- 新增 `MarkRecalled(ctx, id string, recalledAt time.Time) (domain.Message, error)`：
  - `UPDATE messages SET recalled_at=$2 WHERE id=$1 RETURNING ...`
  - 同样更新 head preview 为 "[recalled]"（若是最新消息）

> **head preview 更新**：编辑/撤回的消息可能是也可能不是会话最新消息。只需在"是 head"时更新 preview。用一个条件 UPDATE 或在应用层先查 head seq 再决定。

**决策**：head 更新用一条 SQL 完成——`UPDATE conversations SET last_message_preview = CASE WHEN last_message_seq = $seq THEN $preview ELSE last_message_preview END WHERE id = $convID`。这样即使不是 head 也不影响。

## 4. Service

**Files:** `backend/internal/service/message_service.go`

### MessageStore 接口扩展

```go
type MessageStore interface {
    Insert(...)
    List(...)
    FindByID(...)
    FindByIDs(...)
    UpdateBody(ctx, id, body string, editedAt time.Time) (domain.Message, error)  // NEW
    MarkRecalled(ctx, id string, recalledAt time.Time) (domain.Message, error)    // NEW
}
```

### 新常量与校验

```go
const editRecallWindow = 5 * time.Minute
```

```go
func (s *MessageService) requireOwnRecent(ctx, conversationID, messageID, userID string, now time.Time) (domain.Message, error) {
    // 1. requireMember(conversationID, userID)
    // 2. FindByID(messageID) → 必须在同一 conversation
    // 3. m.SenderID == userID，否则 Forbidden("not your message")
    // 4. now.Sub(m.CreatedAt) <= editRecallWindow，否则 Invalid("edit window expired")
    // 5. 若 m.RecalledAt != nil → Invalid("message already recalled")
}
```

### Edit

```go
func (s *MessageService) Edit(ctx context.Context, conversationID, messageID, userID, body string) (MessageView, error) {
    m := requireOwnRecent(...)
    body = TrimSpace(body); 非空、≤4000 runes
    updated, err := s.messages.UpdateBody(ctx, messageID, body, s.now().UTC())
    // 若 updated.ReplyToMessageID != nil → 重新 hydrate reply preview
    s.broadcast(ctx, MessageView{Message: updated, ...}, "message.edited")
    return view
}
```

### Recall

```go
func (s *MessageService) Recall(ctx context.Context, conversationID, messageID, userID string) (MessageView, error) {
    m := requireOwnRecent(...)
    updated, err := s.messages.MarkRecalled(ctx, messageID, s.now().UTC())
    s.broadcast(ctx, MessageView{Message: updated, ...}, "message.recalled")
    return view
}
```

### broadcast 扩展

现有 `broadcast` 硬编码 `"message.created"`。改为带 eventType 参数：

```go
func (s *MessageService) broadcast(ctx, v MessageView, eventType string)
// Send → "message.created"
// Edit → "message.edited"
// Recall → "message.recalled"
```

`messagePayload` 加字段：

```go
func messagePayload(v MessageView) map[string]any {
    p := map[string]any{
        ...
        "edited_at":   nil,
        "recalled_at": nil,
    }
    if v.Message.EditedAt != nil { p["edited_at"] = v.Message.EditedAt.UTC().Format(RFC3339) }
    if v.Message.RecalledAt != nil { p["recalled_at"] = v.Message.RecalledAt.UTC().Format(RFC3339) }
    return p
}
```

## 5. Handler & Router

**Files:** `backend/internal/handler/message.go`, `backend/internal/handler/router.go`

```go
type editMessageBody struct {
    Body string `json:"body"`
}

func (h *MessageHandler) Edit(w, r, conversationID, messageID)
// decode body → h.Msg.Edit(ctx, conversationID, messageID, UserIDFromContext, body.Body)
// → writeJSON(200, toMessageDTO(view))

func (h *MessageHandler) Recall(w, r, conversationID, messageID)
// h.Msg.Recall(ctx, conversationID, messageID, UserIDFromContext)
// → writeJSON(200, toMessageDTO(view))
```

`toMessageDTO` 需加 `edited_at` / `recalled_at` 字段（与 WS payload 一致）。

**路由：**
```go
mux.Handle("PATCH /v1/conversations/{id}/messages/{messageID}", require(...))
mux.Handle("POST /v1/conversations/{id}/messages/{messageID}/recall", require(...))
```

## 6. 前端

### 类型

```typescript
// api/messages.ts Message
type Message = {
  ...
  edited_at?: string | null;
  recalled_at?: string | null;
};
```

`ChatItem` 继承 Message 字段（types.ts）。

### Realtime

`realtime/index.ts` dispatch switch 加：
```typescript
case 'message.edited':   notify((h) => h.onMessageEdited?.(frame.payload as Message)); break;
case 'message.recalled': notify((h) => h.onMessageRecalled?.(frame.payload as Message)); break;
```
`RealtimeHandlers` 加 `onMessageEdited` / `onMessageRecalled`。`useRealtime` proxy 转发。

### ConversationRoom

```typescript
onMessageEdited: (m) => {
  if (m.conversation_id !== convId) return;
  setMessages((prev) => mergeMessage(prev, toChatItem(m)));
},
onMessageRecalled: (m) => {
  if (m.conversation_id !== convId) return;
  setMessages((prev) => mergeMessage(prev, toChatItem(m)));  // payload 带 recalled_at → 前端渲染占位符
},
```

`mergeMessage` 已按 id upsert，带 `recalled_at` 的消息会被正确合并。

### MessageBubble

- 新增 props `onEdit?` / `onRecall?`
- 自己的消息且 `status === 'sent'` 且 5 分钟内：显示 编辑/撤回 按钮
- 若 `message.recalled_at`：气泡体显示 "已撤回" 占位符（不显示原文）
- 若 `message.edited_at`：meta 区显示 "已编辑" 标记

### 编辑 UI

- 点编辑 → 气泡进入编辑模式（textarea 预填原文 + 保存/取消）
- 保存 → `PATCH .../messages/{id}` → mergeMessage 更新

### 5 分钟窗口

- 前端：`now - created_at <= 5min` 才显示按钮
- 后端强制：超时返回 error

### 会话列表预览

`AppShell` 的 `onMessageEdited` / `onMessageRecalled` 更新对应会话的 preview：
- edited → 新正文
- recalled → `t('chat.recalledPreview')` = "[已撤回]"

## 7. 数据流

### 编辑
```
A 点编辑 → PATCH /v1/conversations/{id}/messages/{mid} {body}
  → MessageService.Edit → 校验(成员/own/5min/未撤回)
  → MessageRepo.UpdateBody → 更新 body+edited_at + head preview（若是head）
  → broadcast message.edited 给所有成员（含A）
  → B 前端 mergeMessage → 显示新正文
  → A 自己的另一设备同步
```

### 撤回
```
A 点撤回 → POST .../messages/{mid}/recall
  → MessageService.Recall → 校验 → MarkRecalled
  → 更新 head preview "[已撤回]"（若是head）
  → broadcast message.recalled
  → 所有客户端显示占位符
```

## 8. 兼容性 / 回滚

- **Migration**：加列是 additive，向后兼容
- **API**：新增端点；message DTO 加 `edited_at`/`recalled_at`（additive）
- **回滚**：移除路由 + 前端按钮；migration 可 down
- **已知取舍**：编辑/撤回不触发已读回执重新计算；撤回的消息 `last_message_seq` 不变（不影响 keyset 分页）
