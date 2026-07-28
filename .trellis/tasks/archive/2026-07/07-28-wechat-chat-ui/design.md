# Design: WeChat-like chat UI

## Architecture

```text
[Conversation list]     [Room shell]
  AppShell side    →      RoomHeader
                          MessageList → MessageBubble (+ ReplyQuote)
                          Composer → EmojiPicker + ReplyBar
                                │
                    api/messages + realtime (WS)
                                │
                    MessageHandler → MessageService → MessageRepo
                                │
                              Hub message.created
```

- **Backend first**：先落地 `reply_to` 契约，前端再接 UI。
- **Frontend extraction**：`ConversationRoom` 逻辑从 `AppShell.tsx` 迁到 `features/chat/`；`AppShell` 保留侧栏创建/列表与 `<Outlet />`。
- 不引入 React Query / 新 UI 库；状态仍以 room 内 hooks + `useState` 为主。

## Backend: reply_to

### Schema

New migration (timestamp after `20260728200000`):

```sql
ALTER TABLE messages
  ADD COLUMN reply_to_message_id UUID NULL
  REFERENCES messages (id) ON DELETE SET NULL;

CREATE INDEX idx_messages_reply_to ON messages (reply_to_message_id)
  WHERE reply_to_message_id IS NOT NULL;
```

Down: drop index + column.

### Domain

```go
type Message struct {
  // existing fields...
  ReplyToMessageID *string // nil = no reply
}
```

API 展示用摘要不进 domain 持久化字段；可在 handler 层组装，或 service 返回 enriched view。推荐：

- Repo 读写 `reply_to_message_id`
- Service `Send` 校验目标后写入 id
- Handler `toMessageDTO` 时若 id 非空则 `FindByID`（或 list 批量）填充 `reply_to`

为减少 N+1：`List` 后收集非空 reply ids → `FindByIDs` → map 填 DTO。单条 Send/broadcast 单次 `FindByID` 即可。

### Contracts

**Send request**

```json
{
  "body": "string",
  "client_msg_id": "string",
  "reply_to_message_id": "uuid-optional"
}
```

省略或 `null` / `""` = 无引用。

**Message DTO / WS payload**

```json
{
  "id": "...",
  "conversation_id": "...",
  "sender_id": "...",
  "body": "...",
  "client_msg_id": "...",
  "seq": 1,
  "created_at": "RFC3339",
  "reply_to": null
}
```

或：

```json
"reply_to": {
  "id": "...",
  "sender_id": "...",
  "body": "..."
}
```

规则：

| 情况 | `reply_to` |
|------|------------|
| 未引用 | `null` 或不输出（推荐显式 `null` 便于前端） |
| 目标存在 | 对象；`body` 按 rune 截断至 **120** 供展示，**存储仍为 id 指向全文** |
| 目标已删（SET NULL 后 id 空） | `null` |
| 目标在他会话 | Send 拒绝，不落库 |

**Send 校验**

1. 现有 body / client_msg_id 规则不变。
2. 若 `reply_to_message_id` 有值：`FindByID`；不存在 → `Invalid("reply target not found")`（或 NotFound；与现有 invalid 风格一致优先 Invalid）。
3. `target.ConversationID != in.ConversationID` → Invalid。
4. 幂等：唯一键冲突返回**首次**消息（含其原 reply），忽略重试 body/reply 差异（与现语义一致）。

### Store interface delta

```go
type MessageStore interface {
  Insert(ctx, m domain.Message) (domain.Message, error)
  List(ctx, conversationID string, beforeSeq int64, limit int) ([]domain.Message, error)
  FindByID(ctx, id string) (domain.Message, error)
  FindByIDs(ctx, ids []string) (map[string]domain.Message, error) // optional; can loop FindByID in tests
}
```

`Insert` SQL 增加 `reply_to_message_id` 列；`FindByClientMsgID` / `List` SELECT 同步。

### Tests

- Service：无 reply 回归；有效 reply；跨会话 reply 失败；不存在 id 失败；幂等。
- mem fake 实现 `FindByID`（及可选 `FindByIDs`）。

## Frontend

### Module layout

```text
frontend/src/features/chat/
  ConversationRoom.tsx   # container: load, ws, send, scroll
  MessageList.tsx
  MessageBubble.tsx
  Composer.tsx           # textarea, send, enter key
  EmojiPicker.tsx        # popover + fixed list
  ReplyBar.tsx           # chip above input
  types.ts               # local view models if needed
  emoji.ts               # constant list
```

`App.tsx` / `AppShell`：export room 从 feature 引入；侧栏可暂留 AppShell。

### Message view model

```ts
type ReplyTo = { id: string; sender_id: string; body: string } | null;

type Message = {
  id: string;
  conversation_id: string;
  sender_id: string;
  body: string;
  client_msg_id: string;
  seq: number;
  created_at: string;
  reply_to?: ReplyTo | null;
};

// local only
type LocalStatus = 'pending' | 'sent' | 'failed';
type ChatItem = Message & { status?: LocalStatus; localKey?: string };
```

### Optimistic send

1. 生成 `client_msg_id`；构造 pending item（临时 id = `local:${client_msg_id}`，`seq` 用 `max+ε` 或始终 append）。
2. Append → scroll bottom → `sendMessage(...)`。
3. 成功：按 `client_msg_id` / `id` 替换为服务端消息，`status: sent`。
4. 失败：`status: failed`，保留 body与 reply 上下文供 Retry。
5. WS `message.created`：若已有同 `client_msg_id` 或 `id` 则 merge/替换，不双插。

### Composer UX

- `textarea`，自动浅增高（max 约 5–8 行）可选，最小一行。
- `onKeyDown`：Enter 无 shift → preventDefault + submit；Shift+Enter 换行。
- ReplyBar：显示 `memberLabel(reply.sender_id)` + 截断 body；清除按钮。
- EmojiPicker：button 切换；选中后 `document` 级或受控 value 在 selectionStart 插入。

### Layout / CSS

- `.workspace` / `.room`：主区 `display:flex; flex-direction:column; min-height:0; height: …`（相对 shell 可用高度，如 `calc(100dvh - header)` 或 workspace `align-items: stretch` + `min-height: 70dvh`）。
- `.msg-list`：`flex:1; overflow-y:auto; max-height:none`（取消固定 28rem 封顶，改由父级约束）。
- `.msg--mine`：右对齐 + 中性深底/深字气泡（**非绿色**）；对方左对齐 + 浅灰/白表面。配色只从现有 `--ink` / `--surface` / `--line` / `--bg` 派生，不引入微信绿。
- 时间分隔：简单规则 — 与上一条间隔 > 5 分钟则插一行时间。
- 焦点环与对比度保留（component-guidelines）。

### Scroll policy

- 初始 load、自己发送、WS 且已接近底部（阈值阈值 e.g. 80px）→ `scrollTo` bottom。
- 用户上翻超过阈值时，WS 新消息不强制跳转（可选未读条；本轮可不做未读条，仅不抢滚动）。

### i18n keys (illustrative)

- `chat.reply`, `chat.cancelReply`, `chat.replyingTo`, `chat.emoji`, `chat.retry`, `chat.sending`, `chat.messagePlaceholder`, time formatting可用 `Intl` 本地化不必全进 JSON。

## Compatibility

- 旧消息 `reply_to_message_id` NULL → `reply_to: null`。
- 旧前端忽略新字段仍可工作；本任务同时升级前后端。
- WS 与 HTTP 字段对齐，避免两套解析。

## Trade-offs

| 选项 | 决定 | 原因 |
|------|------|------|
| body 前缀假引用 vs DB FK | DB FK | 用户要 B；可演进 |
| DTO 嵌摘要 vs 只返 id | 嵌摘要 | 列表无需二次请求；截断 120 |
| 虚拟列表 | 不做 | 本轮 ≤100 条足够 |
| 微信主题色 / 绿色气泡 | 不做 | 用户明确：主题色不参考微信、不用绿色；沿用 minimal 黑白灰 |
| 拆父子任务 | 不拆 | 验收是同一垂直切片 |

## Rollback

- DB：goose down 去掉列（需先停写 reply）。
- 前端：feature 目录可回退到旧 `ConversationRoom`（git revert）。
- 若仅前端回滚：后端多字段无害。

## Risks

- `AppShell.tsx` 体量大，拆分易漏 WS/去重逻辑 → implement 逐步搬迁并手测。
- List N+1：必须批量 hydrate reply。
- 幂等 + 乐观：严格按 `client_msg_id` 去重。
