# 历史消息搜索 — 执行计划

## 前置调研

- [ ] 确认 repo 层是否已有测试文件（`ls backend/internal/repo/*_test.go`），决定测试策略
- [ ] 读 `router.go` 消息路由，确认 `/messages/search` 与 `/messages/{messageID}` 的 ServeMux 优先级（Go 1.22 精确段优先，参照 members/me 先例）
- [ ] 读前端 `ConversationRoom.tsx` 的 `loadMessages` / `setMessages` / `scrollToBottom` 完整逻辑，确认跳转接入点

## 执行步骤

### 1. 后端 repo：Search + ListAround

- [ ] `MessageStore` 接口加 `Search(ctx, conversationID, query string, beforeSeq int64, limit int) ([]domain.Message, error)` 与 `ListAround(ctx, conversationID string, aroundSeq int64, window int) ([]domain.Message, error)`
- [ ] 实现 `Search`：`WHERE conversation_id=$1 AND recalled_at IS NULL AND body ILIKE '%'||$3||'%' AND seq<$4 ORDER BY seq DESC LIMIT $5`
- [ ] 实现 `ListAround`：`seq BETWEEN $2-$3 AND $2+$3 ORDER BY seq ASC`（含撤回）
- **验证**：`go build ./...`

### 2. 后端 service：Search + ListAround

- [ ] `MessageService.Search(ctx, conversationID, userID, query, beforeSeq, limit)`：`requireMember` → `strings.TrimSpace(query)==""` → `apperr.Invalid` → `messages.Search` → `hydrateViews`
- [ ] `MessageService.ListAround(ctx, conversationID, userID, aroundSeq, window)`：`requireMember` → `messages.ListAround` → `hydrateViews`
- **验证**：`go build ./...`

### 3. 后端 handler + 路由

- [ ] `handler.Search`：解析 `q`/`before_seq`/`limit` → `Msg.Search` → `{messages}`（仿 List）
- [ ] `handler.List`：增加 `around_seq` 解析，与 `before_seq` 互斥（同时提供 400），有 around_seq 时走 `Msg.ListAround`
- [ ] `router.go`：`GET /v1/conversations/{id}/messages/search`（精确段优先）
- **验证**：`go build ./...` + 端到端 curl 冒烟

### 4. 后端测试

- [ ] service：`Search`（命中/空 query 400/非成员 404）、`ListAround`（窗口+ACL）
- [ ] handler：`Search`（200/400/404）、around_seq 互斥 400
- [ ] repo：若已有测试文件则补；否则跳过（靠端到端）
- **验证**：`go test ./...`

### 5. 前端 API + realtime 无需改

- [ ] `api/messages.ts`：`searchMessages(id, q, opts)`；`listMessages` 支持 `around_seq`
- **验证**：`npm run typecheck`

### 6. 前端 UI

- [ ] `SearchPanel.tsx`：搜索输入 + 结果列表 + 分页加载更多
- [ ] `ConversationRoom.tsx`：头部搜索按钮 + SearchPanel 挂载 + 点击结果 `loadAround`
- [ ] `MessageList.tsx`：`highlightId` prop + 高亮 class
- [ ] i18n + CSS
- **验证**：`npm run typecheck` + `npm run build`

### 7. 端到端验证

- [ ] 起后端，发多条消息（含关键词 + 撤回一条）
- [ ] 搜索关键词 → 结果正确（不含撤回）
- [ ] 点击结果 → around_seq 加载 + 定位 + 高亮
- [ ] 非成员搜索 → 404；空白 q → 400

## 退出标准（对应 PRD AC）

- [ ] 搜索 API 命中/排除撤回/分页/ACL 全部正确
- [ ] around_seq 跳转加载定位正确
- [ ] 前端搜索面板 + 跳转高亮可用
- [ ] 单测 + typecheck 全绿

## 回滚

- 后端纯增量（新方法 + 新路由 + around_seq 可选参数），无迁移、无破坏性变更。revert 即可。
- around_seq 是现有 List 的可选参数，默认行为不变；不接线前端即退回原样。
