# 历史消息搜索

## Goal

在单个会话内搜索历史消息。用户输入关键词，返回匹配的消息（按时间倒序分页）；点击一条结果跳转到消息在对话中的位置并高亮。仅搜索当前会话（MVP 不做全局跨会话搜索）。

## Requirements

- 会话内按关键词搜索历史消息，返回匹配结果（片段、发送者、时间），按 `seq` 倒序分页。
- 搜索结果**排除已撤回消息**（`recalled_at` 非空的不出现在结果中）。
- 仅会话成员可搜索（ACL：复用 `requireMember`，非成员返回 404）。
- 点击结果**跳转到消息位置**：消息列表加载到该消息附近，滚动定位并高亮。
- 关键词为空/仅空白时拒绝（400）。

## Acceptance Criteria

- [ ] 后端 `GET /v1/conversations/{id}/messages/search?q=...`（`q` 必填、`before_seq`/`limit` 分页）返回匹配消息（`messageDTO` 数组，复用现有 shape）。
- [ ] `q` 空白返回 400；非成员返回 404；`before_seq`/`limit` 非法返回 400。
- [ ] 搜索结果不含已撤回消息。
- [ ] 后端 `around_seq` 加载：`GET .../messages?around_seq=X` 返回 X 附近的消息（X 前后的窗口，按 seq 升序），用于跳转定位。
- [ ] 前端会话头部加搜索入口；搜索面板展示结果列表（片段 + 发送者 + 时间）。
- [ ] 点击搜索结果 → 消息列表加载到该消息附近、滚动定位、高亮该消息。
- [ ] 单元测试：repo Search（ILIKE、撤回排除、分页）、handler Search（400/404）、service Search（ACL）；repo/handler around_seq。
- [ ] `go test ./...`（backend）+ `npm run typecheck`（frontend）通过。
- [ ] 端到端：发若干消息，搜索关键词命中，点击跳转定位。

## Notes

- **搜索实现**：SQL `body ILIKE '%' || $2 || '%'`，不引入全文索引（pg_trgm/pgvector）。MVP 数据量下可接受；后续量大可加索引。
- **搜索范围**：仅当前会话（`conversation_id` 过滤），不做全局跨会话搜索。
- **分页**：搜索结果按 `seq DESC` + `before_seq` 游标（复用 List 模式）；`limit` 上限 100（对齐 List）。
- **around_seq 跳转**：这是本任务的难点。加载窗口 = 目标 seq 前后各 N 条（如各 50），使列表能定位到目标。前端在跳转后把高亮消息 id 传给 `MessageList`。
- **撤回消息在跳转时**：around_seq 加载**包含**已撤回消息（跳转定位需要完整的 seq 序列），但搜索结果**排除**。这是有意的差异，需在测试中明确。
- **前端交互**：搜索面板 + 结果列表 + 点击跳转。复用现有 `MessageBubble` 渲染（或简化片段）。高亮用 CSS class（`msg-item--highlight`）。
- 参考实现：`backend/internal/repo/message_repo.go`（List/keyset 分页）、`backend/internal/service/message_service.go`（requireMember/hydrateViews）、`backend/internal/handler/message.go`（List handler）、`frontend/src/features/chat/ConversationRoom.tsx`（消息加载）、`frontend/src/features/chat/MessageList.tsx`（渲染）、`frontend/src/api/messages.ts`（API 层）。
