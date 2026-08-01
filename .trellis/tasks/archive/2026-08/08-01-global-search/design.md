# 全局搜索与关键词高亮 — 技术设计

## 1. 数据流

### 全局搜索
```
前端全局搜索框 → GET /v1/search/messages?q=xx&cursor=&limit=
  → handler.GlobalSearch → service.GlobalSearch
      → repo.GlobalSearch (JOIN conversation_members 限定 ACL + conversations 取标题)
  → messageSearchResultDTO[]（含 conversation_id / conversation_title / 消息字段）
  → 全局搜索面板（会话标题 + 片段 + 发送者 + 时间）
```

### 关键词高亮（纯前端）
```
SearchPanel / 全局搜索面板渲染结果时 → highlight(query, body)
  → body 中匹配段包 <mark>（大小写不敏感，HTML 转义）
```

## 2. 契约

### 全局搜索 HTTP

`GET /v1/search/messages`

Query params:
- `q`（必填，trim 后非空，否则 400）
- `cursor`（可选，`created_at|id` 复合游标；缺省 = 最新）
- `limit`（可选，默认 50，上限 100）

响应 200:
```json
{
  "messages": [
    {
      "conversation_id": "...",
      "conversation_title": "群名" | null,   // null = DM，前端推断
      "id": "...", "sender_id": "...", "body": "...",
      "client_msg_id": "...", "seq": 7, "created_at": "...",
      "edited_at": null, "recalled_at": null, "reply_to": null
    }
  ],
  "next_cursor": "2026-08-01T12:00:00Z|msg-id" | null
}
```

- **游标**：跨会话 `seq` 不唯一，用 `(created_at, id)` 复合游标。`WHERE (created_at, id) < ($1, $2)` 分页，`ORDER BY created_at DESC, id DESC`。`next_cursor` = 最后一条的 `created_at + "|" + id`；不足一页时 null。
- **错误**：`q` 空白 → 400；非法 cursor/limit → 400；未认证 → 401。

### 关键词高亮

纯前端工具函数（`features/chat/searchHighlight.ts`）：
```ts
export function highlightQuery(body: string, query: string): ReactNode[];
```
- 大小写不敏感匹配；`<mark>` 包裹命中段；body 中非命中文本与 `&<>` 转义后原样输出。
- 应用于：全局搜索面板 + 会话内 SearchPanel 的结果正文。

## 3. 后端改动

| 文件 | 改动 |
|------|------|
| `internal/repo/message_repo.go` | `GlobalSearch(ctx, userID, query string, cursor *searchCursor, limit int) ([]GlobalSearchRow, next *searchCursor, error)`：`SELECT m.id, m.conversation_id, m.sender_id, m.body, m.client_msg_id, m.seq, m.created_at, m.reply_to_message_id, m.edited_at, m.recalled_at, c.title FROM messages m JOIN conversation_members cm ON cm.conversation_id=m.conversation_id AND cm.user_id=$1 JOIN conversations c ON c.id=m.conversation_id WHERE m.recalled_at IS NULL AND m.body ILIKE '%'||$2||'%' [AND (m.created_at, m.id) < ($3,$4)] ORDER BY m.created_at DESC, m.id DESC LIMIT $5` |
| `internal/service/message_service.go` | `GlobalSearch(ctx, userID, query string, cursor *..., limit int)`：trim 空拒 400 → `repo.GlobalSearch` → 组装含会话标题的 view |
| `internal/handler/search.go`（新） | `GlobalSearch(w,r)`：解析 `q`/`cursor`/`limit` → `Msg.GlobalSearch` → `{messages, next_cursor}`；`searchResultDTO`（消息字段 + `conversation_id` + `conversation_title`） |
| `internal/handler/router.go` | `GET /v1/search/messages` |

> `searchCursor` 在 service 层定义为小结构体 `{CreatedAt time.Time; ID string}`，repo 负责解析/序列化 `created_at|id` 字符串。handler 解析 `cursor` 参数传入 service。

### DTO 复用
`searchResultDTO` 复用消息字段（`toMessageDTO` 的字段），加 `conversation_id` + `conversation_title`。可提取公共转换，或直接构造（消息字段少，直接构造更简）。

## 4. 前端改动

| 文件 | 改动 |
|------|------|
| `src/api/messages.ts` | `globalSearchMessages(query, opts?)` → `GET /v1/search/messages` |
| `src/features/chat/searchHighlight.tsx`（新） | `highlightQuery(body, query): ReactNode[]` |
| `src/features/chat/SearchPanel.tsx` | 结果正文用 `highlightQuery` 高亮 |
| `src/features/chat/GlobalSearchPanel.tsx`（新） | 全局搜索面板：会话标题 + 高亮片段 + 发送者 + 时间；点击结果 `navigate(/app/c/${conversation_id})` |
| `src/app/AppShell.tsx` | 会话列表顶部加全局搜索入口（按钮 → 展开 GlobalSearchPanel 或跳 `/search` 页） |
| `src/styles/index.css` | 全局搜索面板样式 + `<mark>` 高亮样式 |
| i18n | `chat.globalSearch*` 文案（zh-CN/en） |

### 跳转
全局搜索结果点击 → `navigate(/app/c/${conversation_id})` → ConversationRoom 打开该会话（加载最新消息）。**不做**跳到具体 seq（MVP 只跳会话；跳到消息需 around_seq 组合，可后续增强）。

## 5. 测试

### service（message_service_test.go）
- `TestGlobalSearch`：跨会话命中、ACL 限定（仅所属会话）、排除撤回
- `TestGlobalSearchBlankQuery`：400
- `TestGlobalSearchCursor`：游标分页（mock 需支持）

### handler（search_test.go 新文件）
- `TestGlobalSearchHandler`：200 命中 + next_cursor
- `TestGlobalSearchHandlerBlank`：400
- `TestGlobalSearchHandlerBadCursor`：400

### 前端（无测试框架？确认）
- `highlightQuery` 若可测则补纯函数测试；项目无 vitest 则跳过（typecheck 覆盖）

## 6. 明确不做（Out of scope）

- 全局搜索的**跳转到具体消息**（MVP 只跳会话，around_seq 组合后续做）。
- 全文索引（pg_trgm/pgvector）、中文分词。
- 搜索结果分组（按会话聚合）——扁平列表 + 会话标题即可。
- 高亮命中次数的统计/计数。

## 7. 风险与决策点

- **跨会话游标**：`(created_at, id)` 游标保证分页稳定。风险：同一毫秒多条消息时靠 id 破平（UUID 无序，但 `id < $2` 字符串比较是确定性的，可能跳过同毫秒消息）。替代：`seq` 全局化（大改，不做）。接受当前方案的确定性分页。
- **ACL**：join `conversation_members` 是权威限定。用户退出会话后历史消息不再可搜（符合直觉）。
- **高亮转义**：body 是用户输入，`<mark>` 插入前必须转义非命中文本，防 XSS。
- **性能**：`ILIKE '%q%'` 跨全表扫描，按用户 join 后数据量 = 用户所有会话消息。MVP 可接受；量大迁移 pg_trgm。
