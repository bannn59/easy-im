# 好友驱动开聊（替代邮箱建会话）

## Goal

在已有好友关系的前提下，用「选好友开 1:1」替代「输入邮箱建会话」；**完全移除** `member_emails` 创建入口，使主路径私聊不再对任意邮箱开放。

## Background

- 父任务：`07-29-friends-chat`。
- 依赖子任务 `07-29-friends-relation` **已归档交付**：`friend_requests` / `friendships`、`/v1/friends*`、Friends 页。
- 现状仍在：`POST /v1/conversations` + `member_emails`；`AppShell` 邮箱创建表单；**无** get-or-create DM。

## Dependencies

- **已满足**：`07-29-friends-relation` 可查询「已是好友」与好友列表。
- 依赖写在本文件，不靠目录树位置暗示。

## Confirmed Facts

- 创建入口：`backend/internal/handler/conversation.go` + `service.ConversationService.Create` + `frontend` `createConversation({ member_emails })`。
- 好友边：`friendships` 无向 canonical pair；`FriendService` / repo 可复用或扩展 `AreFriends`。
- 列表/消息/WS **不**因本任务改为「发消息前校验好友」。

## Requirements

- R1. 新开聊入口：从好友列表（或等价「与某好友开始聊天」）发起 1:1。
- R2. 服务端在**新建/get-or-create 开聊**时校验双方已是好友；非好友拒绝。
- R3. API 与前端 **移除** `member_emails` 创建会话能力（请求体、客户端表单、相关文案）。
- R4. **历史会话保留可聊**：改造前已存在会话仍可列表、读历史、发消息；发消息路径**不**强制好友。
- R5. **Get-or-create 唯一 1:1**：对好友 B 开聊时，若已存在成员集合恰好为 `{A,B}` 的会话则复用；否则创建。
- R6. 历史已有多条 A–B 二人会话：复用**一条**——优先「最近有消息」（`last_message_at` 最新），若都无消息则最近创建；**不做**合并迁移。
- R7. 删除会话 / 清空历史 / 列表删除态多端同步：**不做**（父级已定）。

## Out of Scope

- 好友请求/同意/列表（已由 relation 交付）
- 群聊创建 / 多成员会话产品化
- 拉黑、删除好友
- 历史非好友会话只读化/隐藏
- 多条 1:1 合并迁移
- 删会话、清空记录、列表态多端同步

## Acceptance Criteria

- [ ] AC1. 好友 A、B 可从好友入口进入 1:1 并收发消息。
- [ ] AC2. 非好友无法通过新开聊 API/UI 创建 1:1。
- [ ] AC3. 旧 `member_emails` 创建不可用（接口移除或稳定拒绝）；前端无邮箱建会话表单。
- [ ] AC4. 改造前已存在会话：成员仍可列表可见、读历史、发消息（不因「彼此还不是好友」被拦）。
- [ ] AC5. 对同一对好友连续两次「开聊」，得到**同一** `conversation id`，不新增平行 1:1。

## Key Decisions

| 决策 | 选择 |
|------|------|
| 邮箱建会话 | **完全移除** |
| 历史非好友会话 | **保留可聊**（只拦新建） |
| 1:1 | **Get-or-create**；多历史条时复用最近有消息 / 否则最近创建 |
| 删会话 / 列表态多端 | **不做** |
| 规划形态 | PRD + jsonl（与 relation 一致；无独立 design.md 强制） |

## Open Questions

- （无阻塞项。）

## Notes

- 实现建议形态（非 PRD 强制 API 名）：例如 `POST /v1/friends/{id}/conversation` 或 `POST /v1/conversations` body 改为 `peer_user_id`；须好友校验 + get-or-create。
- Start 前 jsonl 已固化；用户确认本摘要后方可 `task.py start`。
