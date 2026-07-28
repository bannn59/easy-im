# Conversation list: last message preview and unread

## Goal

把 `/app` 会话侧栏从「标题 + id 片段」升级为接近微信列表的 **表面完整度**：最后一条预览、相对时间、未读角标、按最后活动排序，并在收到实时消息时更新对应行。

用户价值：进入工作区第一眼就能判断「谁在说话、哪边有新消息」。

## Background

- 功能图（t6 refresh）：M0–M4 已完成；P5.a 已读回执等仍为 not_found。
- 北极星 **B（微信表面）** 第一刀；**不是** 给对方看的已读产品化（P5.a 完整版）。
- 已归档 `wechat-chat-ui`：room 成熟；侧栏仍简陋。
- 证据：`conversations` 无 last_message 列；`conversation_members` 无 `last_read_seq`；list 按 `updated_at`；FE list 无预览/未读；WS 仅在 room merge。

## Grill decisions (authoritative)

| Decision | Choice |
|----------|--------|
| 北极星 | B 微信表面 |
| 切片 | 预览 + 未读同一切片 |
| 未读 | **仅自己侧**；无对方已读 |
| 清未读 | 打开会话成功 load 后 mark 到 max seq |
| 自己发送 | 推进自己 `last_read_seq` |
| 未读数字 | **只计对方消息** |
| last message | **会话表冗余**，发送事务更新 |
| 侧栏实时 | 复用 `message.created` patch |
| 行 UI | 标题、预览、相对时间、角标；按 last 活动排序；群预览带发送者前缀 |
| 预览 | **仅文本截断**（无 type 字段） |

## Requirements

### R1 — Conversation head

- 发送落库时更新 head：`last_message_at`、`last_message_seq`、`last_message_preview`（截断）、`last_message_sender_id`。
- `GET /v1/conversations` 返回 head + 每用户 `unread_count`；排序按最后消息时间降序（无消息回退 `updated_at`/`created_at`）。

### R2 — Self-only unread

- members 增加 `last_read_seq`（默认 0）。
- `unread_count` = 对方消息且 `seq > last_read_seq` 的条数。
- 进房成功 load 后 mark-read → 清零。
- 自己发送成功 → 提升自己的 `last_read_seq`。

### R3 — Mark-read API

- `POST /v1/conversations/{id}/read`（可选 body `seq`，默认 head/max）。
- 仅成员；**不**广播已读 WS 事件。

### R4 — Sidebar UI

- 预览、相对时间、未读角标；群前缀发送者短名；DM 可用 body /「你: …」。
- 选中态保留。

### R5 — Realtime list patch

- Workspace 级处理 `message.created`：更新 preview/时间/排序；`sender≠me` 且不在该 room → 未读 +1。
- 不强制新事件类型。

### R6 — i18n

- 新文案 en / zh-CN。

## Out of scope

对方已读/双勾、typing、presence、媒体与 type 字段、历史上翻、通讯录、MQ/推送、置顶/免打扰/删除、列表搜索、跨设备 read WS 同步。

## Constraints

- 回归：auth、发消息、room、reply_to、WS。
- 短事务；发送后 commit 再 hub push。
- List 避免未读 N+1 失控。
- 迁移可逆；无微信绿。

## Acceptance Criteria

- [ ] AC1：对方发送后列表出现预览与较新时间，排序靠前。
- [ ] AC2：未打开会话时未读只含对方消息。
- [ ] AC3：打开并成功加载后未读清零。
- [ ] AC4：自己发送不增加自己的未读；预览可更新。
- [ ] AC5：挂在 `/app` 时 WS 更新侧栏预览/未读，无需整页刷新。
- [ ] AC6：群预览带发送者短前缀。
- [ ] AC7：无对方已读 UI；消息收发回归可用。
- [ ] AC8：相关 `go test` 与 `npm run build` 通过；migrate up 成功。

## Notes

- 复杂任务：`design.md` + `implement.md` + jsonl；用户确认最终规划前不 `task.py start`。
