# 历史消息搜索 — 技术设计

## 1. 数据流

### 搜索
```
前端搜索框 → GET /v1/conversations/{id}/messages/search?q=xx&before_seq=&limit=
  → handler.Search → service.Search (requireMember ACL)
      → repo.Search (ILIKE + 排除撤回 + seq<before DESC)
  → messageDTO[]  → 搜索面板结果列表（片段+发送者+时间）
```

### 跳转
```
点击结果 (msg seq=X)
  → GET /v1/conversations/{id}/messages?around_seq=X
  → repo.ListAround (X 前后各 N 条，seq 升序，含撤回消息——定位需要完整序列)
  → 前端 setMessages(结果) → 滚动到 seq=X → 高亮 id
```

## 2. 契约

### 搜索 HTTP

`GET /v1/conversations/{id}/messages/search`

Query params:
- `q` (必填，trim 后非空，否则 400)
- `before_seq` (可选，默认 0=最新；负数 400)
- `limit` (可选，默认 50，上限 100；非法 400)

响应 200:
```json
{ "messages": [ { messageDTO }, ... ] }  // 按 seq DESC，排除撤回
```

### around_seq 加载 HTTP

`GET /v1/conversations/{id}/messages?around_seq=X`

- `around_seq` 存在时，返回 X 前后各 `window` 条（默认 50），按 seq 升序，**含撤回消息**。
- 与现有 `before_seq` 分页**互斥**：同时提供返回 400。无 `around_seq` 时走原 `before_seq` 逻辑。
- 响应同 List：`{ "messages": [...] }`。

> 决策：`around_seq` 复用现有 List 路由（新增可选参数），而非新端点。前端通过 `listMessages(id, { around_seq: X })` 调用。

### 消息 DTO
复用 `messageDTO`（不变），前端已能渲染。

## 3. 后端改动

| 文件 | 改动 |
|------|------|
| `internal/repo/message_repo.go` | `MessageStore` 接口 + `Search(ctx, conversationID, query string, beforeSeq int64, limit int) ([]domain.Message, error)`：`WHERE conversation_id=$1 AND recalled_at IS NULL AND body ILIKE '%'||$3||'%' AND seq<$4 ORDER BY seq DESC LIMIT`；`ListAround(ctx, conversationID, aroundSeq int64, window int) ([]domain.Message, error)`：`seq BETWEEN $2-$3 AND $2+$3 ORDER BY seq ASC`（窗口±window） |
| `internal/service/message_service.go` | `Search(ctx, conversationID, userID, query string, beforeSeq int64, limit int) ([]MessageView, error)`：`requireMember` → trim 空拒 400 → `messages.Search` → `hydrateViews`；`ListAround(ctx, conversationID, userID, aroundSeq int64, window int) ([]MessageView, error)`：`requireMember` → `ListAround` → `hydrateViews` |
| `internal/handler/message.go` | `Search(w,r,conversationID)`：解析 `q`/`before_seq`/`limit` → `Msg.Search` → `{messages}`；`List` 增加 `around_seq` 分支 |
| `internal/handler/router.go` | `GET /v1/conversations/{id}/messages/search`（在 `{id}/messages` 前注册或精确段优先——Go 1.22 ServeMux 精确段 `/search` 优先于 `/{messageID}` 通配；当前消息子路由是独立注册，需确认无冲突） |

> 路由注意：现有 `GET /v1/conversations/{id}/messages` 是独立 route，`search` 是同一路径下的细分。注册 `GET .../messages/search` 时确保 ServeMux 不把它解析成 `{id}` 通配的 `messageID`（`search` 是字面量段，优先级高于通配，Go 1.22 行为——参照现有 `members/me` vs `members/{userID}` 的先例）。

## 4. 前端改动

| 文件 | 改动 |
|------|------|
| `src/api/messages.ts` | `searchMessages(conversationId, q, opts?)`；`listMessages` 增加 `around_seq` 参数 |
| `src/features/chat/ConversationRoom.tsx` | 会话头部加搜索按钮（放大镜）；`SearchPanel` 状态 + 结果 state；点击结果 → `loadAround(seq)` → setMessages + scrollTo(seq) + highlightId |
| `src/features/chat/MessageList.tsx` | `highlightId?: string` prop，命中消息加 `msg-item--highlight` class |
| `src/features/chat/SearchPanel.tsx`（新） | 搜索输入 + 结果列表（复用 MessageBubble 或简化片段）；分页「加载更多」 |
| `src/styles/index.css` | `msg-item--highlight`（背景高亮）、搜索面板样式 |
| i18n | `chat.search*` 文案（zh-CN/en） |

### 跳转流程细节

1. 用户点结果（seq=X, id=M）。
2. `loadAround(X)`：`listMessages(id, { around_seq: X, limit: 100 })` → setMessages（覆盖当前列表）。
3. `requestAnimationFrame` 后，找到 seq=X 的元素，`scrollIntoView` 并 setHighlightId(M)。
4. 高亮在几秒后自动清除（可选，简化 MVP 常亮直到下次跳转/滚动）。
5. 实时新消息仍走 `onMessageCreated` 合并（`mergeMessage` 按 seq 排序，兼容）。

## 5. 测试

### repo（message_repo 无现有测试文件？确认后定）
- `Search`：命中关键词、排除撤回、分页 before_seq、limit 上限
- `ListAround`：返回窗口内消息、seq 升序、包含撤回

> repo 层现有测试可能为空（`go test` 显示 `[no test files]`），若如此，需补一个用 pgxmock 或真实 DB 的测试。真实 DB（docker compose postgres）可做集成测试，或简化：handler/service 用内存 store 断言，repo SQL 靠端到端验证。

### service（message_service_test.go，内存 store）
- `Search` 命中/空 query 400/非成员 404
- `ListAround` 窗口返回 + ACL

### handler（message_test.go / group_test.go 类似）
- `Search` 200 命中 / q 空白 400 / 非成员 404
- `around_seq` 与 `before_seq` 互斥 400

## 6. 明确不做（Out of scope）

- 全局跨会话搜索、全文索引（pg_trgm/pgvector）、搜索关键词高亮、跨会话结果聚合。
- 搜索结果的无限滚动加载（MVP 用「加载更多」按钮或一次性 50 条）。
- 跳转时的消息乐观合并/重连恢复。

## 7. 风险与决策点

- **`around_seq` vs 新建端点**：复用 List 加可选参数最简，前端 API 也最简单。风险：语义混杂（一个端点两种模式）。用参数互斥校验缓解。
- **搜索性能**：ILIKE `%q%` 无法走 B-tree 索引，全表扫描。MVP 数据量（个人 IM）可接受；后续量大迁移 pg_trgm GIN 索引（需迁移 + 触发器维护），本任务不做。
- **跳转窗口**：`around_seq` 返回 X±50 条。若 X 离列表头部很远，用户只能看到局部窗口而非完整历史——MVP 接受（定位是目的）。
- **撤回消息在 around_seq 包含、search 排除**：有意的语义差异，测试中明确。
