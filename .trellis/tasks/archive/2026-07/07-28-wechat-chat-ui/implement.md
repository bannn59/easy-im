# Implement: WeChat-like chat UI

## Ordered checklist

### Phase A — Backend reply_to

1. [ ] 新增 goose migration：`reply_to_message_id` + partial index；Down 完整。
2. [ ] `domain.Message` 增加 `ReplyToMessageID *string`。
3. [ ] `MessageRepo`：Insert/List/FindByClientMsgID 读写新列；新增 `FindByID`（及 `FindByIDs` 或 list 后批量）。
4. [ ] `MessageStore` 接口 + `message_service` mem fake 同步；`SendMessageInput` 增加可选 reply id。
5. [ ] `Send` 校验同会话目标；`broadcast` / handler DTO 输出 `reply_to` 摘要（body 截断 120 runes）。
6. [ ] `List` 批量填充 `reply_to`。
7. [ ] 单测：有效 reply、跨会话、缺失目标、无 reply 回归、幂等。
8. [ ] 验证：`cd backend && go test ./internal/service/ ./internal/handler/ ...`（至少 service）；本地 migrate up 若有 DB。

### Phase B — Frontend API / realtime types

9. [ ] `api/messages.ts`：`Message` / `sendMessage` 请求体支持 `reply_to` / `reply_to_message_id`。
10. [ ] 确认 `realtime` 解析 payload 兼容新字段（类型扩展即可）。

### Phase C — Chat feature UI

11. [ ] 建 `features/chat/`：从 `AppShell.tsx` 迁出 `ConversationRoom` 及相关 helper。
12. [ ] 布局 CSS：room 三层 flex；msg-list 吃满剩余高度；去掉仅 28rem 封顶依赖。
13. [ ] `MessageBubble` + 时间分隔 + 头像占位 + `reply_to` 块。
14. [ ] `Composer`：textarea、Enter/Shift+Enter、发送中态。
15. [ ] 乐观更新 + failed retry + 与 WS 去重。
16. [ ] `EmojiPicker` 固定列表插入。
17. [ ] `ReplyBar` + 气泡/列表「回复」操作。
18. [ ] i18n en + zh-CN 新键；侧栏选中态可顺手加强（非必须 AC）。
19. [ ] `AppShell` / routes 接线；删除死代码（仅本任务产生的）。

### Phase D — Verify

20. [ ] `cd frontend && npm run build`（或 project 既有 typecheck）。
21. [ ] 手测路径：登录 → 建会话 → 发文本 → 表情 → 回复 → 刷新后引用仍在 → 第二用户/两标签 WS。
22. [ ] 对照 `prd.md` AC1–AC9 勾选。

## Validation commands

```bash
# backend
cd backend && go test ./internal/service/ -count=1
cd backend && go test ./... -count=1

# migrate (dev)
export DATABASE_URL='postgres://easyim:easyim@localhost:5433/easyim?sslmode=disable'
cd backend && go run ./cmd/migrate up

# frontend
cd frontend && npm run build
```

## Risky files / rollback points

| 区域 | 文件 | 风险 |
|------|------|------|
| 拆分 UI | `frontend/src/app/AppShell.tsx` | 易丢 WS/去重；迁完立即手测收发 |
| 消息 SQL | `backend/internal/repo/message_repo.go` | 列遗漏导致 scan 失败 |
| 契约 | handler DTO + broadcast map | HTTP/WS 不一致 |
| 样式 | `frontend/src/styles/index.css` | 高度链 min-height:0 易滚不动 |

Rollback：git revert 本任务提交；DB `migrate down` 一版（确认无生产依赖 reply）。

## jsonl manifests

- `implement.jsonl` / `check.jsonl`：frontend 目录/组件/state + backend database/realtime/errors + 本任务 design/prd。

## Not before start

- 用户批准本规划摘要后才 `task.py start`。
- start 后 Phase 2 按 A→B→C→D；布局可对齐微信，**主题色禁止绿色/微信配色**。
