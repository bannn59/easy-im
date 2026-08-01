# 全局搜索与关键词高亮 — 执行计划

## 前置调研

- [ ] 确认 `memMsg` mock 是否需扩展 `GlobalSearch`（会，`MessageStore` 接口将新增方法）
- [ ] 读 `handler/message.go` 的 `toMessageDTO`，决定 `searchResultDTO` 复用方式
- [ ] 确认前端是否已有测试框架（`package.json` scripts / vitest）——决定 `highlightQuery` 测试策略

## 执行步骤（后端先行）

### 1. 后端 repo：GlobalSearch

- [ ] `MessageStore` 接口加 `GlobalSearch(ctx, userID, query string, cursor *searchCursor, limit int) ([]domain.Message, *searchCursor, error)` — 但 cursor 类型在 repo/service 共享，放 domain 或 service 定义
- [ ] 实现 SQL：JOIN conversation_members（ACL）+ conversations（标题）+ ILIKE + 排除撤回 + `(created_at,id)` 游标分页
- [ ] 定义 `searchCursor`（`CreatedAt time.Time; ID string`）与解析/序列化
- **验证**：`go build ./...`

### 2. 后端 service：GlobalSearch

- [ ] `GlobalSearch(ctx, userID, query string, cursor, limit)`：trim 空 400 → repo → 组装含会话标题的 view
- **验证**：`go build ./...`

### 3. 后端 handler + 路由

- [ ] `search.go`：`searchResultDTO` + `GlobalSearch` handler（解析 q/cursor/limit → `{messages, next_cursor}`）
- [ ] `router.go`：`GET /v1/search/messages`
- **验证**：`go build ./...` + curl 冒烟

### 4. 后端测试

- [ ] service：`GlobalSearch` 命中/ACL/排除撤回/空白 400/cursor 分页
- [ ] handler：`GlobalSearch` 200/400（空白、坏 cursor）
- [ ] 更新 `memMsg`（service）与 `memMsgStore`（handler）、`memMessageStore`（fanout）mock 加 `GlobalSearch`
- **验证**：`go test ./...`

### 5. 前端：高亮工具 + API

- [ ] `searchHighlight.tsx`：`highlightQuery(body, query)`（转义 + `<mark>`）
- [ ] `api/messages.ts`：`globalSearchMessages(query, opts)`
- **验证**：`npm run typecheck`

### 6. 前端：全局搜索 UI

- [ ] `GlobalSearchPanel.tsx`：输入 + 结果（会话标题 + 高亮片段 + 发送者 + 时间）+ 加载更多 + 点击跳会话
- [ ] `AppShell.tsx`：会话列表顶部搜索入口
- [ ] `SearchPanel.tsx`：结果正文用 `highlightQuery` 高亮
- [ ] i18n + CSS
- **验证**：`npm run typecheck` + `npm run build`

### 7. 端到端验证

- [ ] 两个会话各有命中消息，全局搜索返回跨会话结果
- [ ] ACL：非所属会话结果不出现（用第三个用户验证）
- [ ] 关键词高亮在前端渲染生效（DOM 检查）

## 退出标准（对应 PRD AC）

- [ ] 全局搜索跨会话 + ACL 限定 + 排除撤回 + 游标分页全部正确
- [ ] 关键词高亮在全局搜索与会话内搜索均生效
- [ ] 前端全局搜索面板 + 点击跳转会话可用
- [ ] 单测 + typecheck 全绿

## 回滚

- 后端纯增量（新端点 + 新 repo 方法），无迁移、无破坏性变更。revert 即可。
- 前端高亮是纯渲染增强，不接不影响搜索功能。
