# 全局搜索与关键词高亮

## Goal

在**所有会话**中搜索当前用户的历史消息（全局搜索），结果展示所属会话信息；并在搜索结果的正文中**高亮**匹配关键词。全局搜索是现有会话内搜索（`/messages/search`）的自然扩展，关键词高亮同时作用于全局搜索与会话内搜索的结果。

## Requirements

- 全局搜索：跨当前用户所属的所有会话搜索消息，按时间倒序分页。
- **ACL**：仅返回用户**所属会话**的消息（通过 `conversation_members` join 限定），非所属会话绝不泄露。
- 结果需携带**会话上下文**（会话 id + 标题），前端据此展示结果来源并可跳转到对应会话。
- 排除已撤回消息（沿用现有 Search）。
- **关键词高亮**：搜索结果正文中高亮匹配的关键词（`<mark>`），同时应用到会话内搜索与全局搜索。
- 空白关键词拒绝（400，沿用现有）。

## Acceptance Criteria

- [ ] 后端 `GET /v1/search/messages?q=...`（`q` 必填、分页）返回跨会话匹配结果，每条含 `conversation_id`、`conversation_title`、消息字段。
- [ ] ACL：仅返回用户所属会话的消息；非所属会话结果不出现（用 SQL join `conversation_members` 限定）。
- [ ] 结果按时间倒序分页（跨会话 `seq` 不唯一，用 `created_at + id` 作游标）。
- [ ] 关键词高亮：全局搜索与会话内搜索的结果正文中，匹配片段用 `<mark>` 高亮。
- [ ] 前端全局搜索 UI：会话列表页顶部搜索入口，展示结果（会话标题 + 片段 + 发送者 + 时间），点击跳转到对应会话。
- [ ] 单元测试：repo 全局搜索（ACL 限定、分页、排除撤回）、handler（400/404）、高亮工具函数。
- [ ] `go test ./...`（backend）+ `npm run typecheck`（frontend）通过。
- [ ] 端到端：两个会话各有命中消息，全局搜索返回跨会话结果且 ACL 正确；关键词高亮生效。

## Notes

- **分页游标**：跨会话搜索不能用 `seq`（每个会话独立计数）。用 `created_at + id` 复合游标（`cursor` = 上一页最后一条的 `created_at` + `id`，`WHERE (created_at, id) < ($1, $2)`）。
- **ACL SQL**：`FROM messages m JOIN conversation_members cm ON cm.conversation_id = m.conversation_id AND cm.user_id = $user`，天然限定所属会话。
- **会话标题**：结果 join `conversations` 取 `title`；DM 会话 title 为 null，前端用成员推断（与列表一致）。
- **关键词高亮**：纯前端。`highlight(query, body)` 返回 ReactNode 数组（`<mark>` 包裹匹配段）。大小写不敏感匹配。注意 HTML 转义（body 是用户输入）。应同时应用于全局搜索结果与现有会话内搜索面板。
- **前端全局搜索入口**：会话列表页（AppShell）顶部加搜索框，或独立页面 `/search`。倾向独立搜索面板/页面，避免污染会话列表布局。
- 参考实现：现有 `Search`（repo/service/handler）、`SearchPanel.tsx`（会话内搜索 UI）、`conversationListTitle`（AppShell 标题推断）、`toConversationDTO`（会话 DTO）。
