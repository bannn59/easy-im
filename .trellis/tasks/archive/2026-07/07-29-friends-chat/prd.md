# 好友关系驱动聊天（替代邮箱建会话）

## Goal

把「发起私聊」从「输入对方邮箱建会话」改为「先建立好友关系，再从好友开聊」，使 1:1 聊天建立在已确认的社交关系上。

本任务是 **父任务**：只拥有需求总集、子任务地图、跨子任务验收与最终集成审视；**不直接作为实现目标**。实现落在两个子任务上。

## Background

- 产品已有：注册/登录、会话列表、HTTP 消息、WS 实时、会话 last-message preview / self-only unread。
- 早期 feature-map 将「通讯录 / 好友」记为 `not_found`；roadmap 把「复杂好友图谱」放远期。本父任务要的是 **薄 MVP 关系 + 开聊入口切换**，不是图谱。

## Confirmed Facts（代码勘察）

### 建会话（邮箱）

- `POST /v1/conversations` body：`title?` + `member_emails: string[]`
  - Handler `backend/internal/handler/conversation.go`
  - Service `backend/internal/service/conversation_service.go`：`FindIDsByEmails`；未知邮箱报错
  - Repo `backend/internal/repo/conversation_repo.go`：事务插 `conversations` + `conversation_members`
- **无** get-or-create DM；同一对用户可多次建 1:1
- 前端 `frontend/src/api/conversations.ts` + `AppShell.tsx` 用邮箱发起创建

### 好友

- **不存在**：无 friends / friend_requests 表、domain、API、前端通讯录

### 用户标识

- `users` 仅有唯一 `email` + UUID `id`；**无 username / 展示名搜索字段**

### 可复用

- 鉴权、会话成员 ACL、消息与实时链路；邮箱 lookup 可复用于「按邮箱发好友请求」（语义不再是直接入会话）

## Task Map

| 子任务 | 目录 | 交付 | 依赖 |
|--------|------|------|------|
| 好友关系 MVP | `07-29-friends-relation` | 按邮箱发请求、同意/拒绝、好友与待处理列表（API + 最小 UI） | 无 |
| 好友驱动开聊 | `07-29-friends-open-chat` | 从好友开 1:1；**完全移除** `member_emails` 建会话 | **必须在 relation 可验收后**（写在子任务 PRD，非树位置暗示） |

## Requirements（跨子任务 / 父级）

- R1. 未建立好友关系时，用户不能再通过「输入邮箱」直接创建与对方的会话。
- R2. 已是好友的双方，能从好友入口发起（或进入）1:1 聊天。
- R3. 好友关系：按邮箱发请求 → 对方同意后互为好友；可拒绝；可独立验收列表与 pending。
- R4. 开聊改造可独立验收：主路径不再暴露/接受 `member_emails` 建会话。
- R5. `POST` 创建会话不再接受邮箱成员列表（open-chat 落地）；新 1:1 仅能针对已是好友的用户发起。

## Out of Scope（父级默认）

- 复杂好友图谱、推荐好友、通讯录同步
- 音视频、消息全站搜索
- 群聊产品化 / 多成员创建新路径（本轮移除邮箱创建后，**本父任务不另做群聊创建**；若需群聊另开任务）
- 屏蔽/拉黑、备注名、好友分组

## Acceptance Criteria（集成）

- [ ] AC-P1. 两名测试用户可完成：按邮箱加好友 → 同意 → 从好友开聊 → 收发消息。
- [ ] AC-P2. 客户端与 API 均无法再通过 `member_emails`（或等价邮箱建会话）创建新会话。
- [ ] AC-P3. 非好友不能通过新开聊入口创建 1:1。
- [ ] AC-P4. `friends-relation` 与 `friends-open-chat` 各自 AC 均满足；父任务不替代子任务验收。

## Key Decisions

| 决策 | 选择 | 备注 |
|------|------|------|
| 任务结构 | **父 + 2 子任务** | 2026-07-29 |
| 加好友发现方式 | **仅按注册邮箱发请求** | 2026-07-29 |
| 好友是否双向同意 | **请求 → 同意后才是好友；可拒绝** | 2026-07-29 |
| 邮箱建会话入口 | **完全移除 `member_emails` 创建** | 2026-07-29；群聊创建不在本父任务范围 |
| 历史非好友会话 | **保留可聊**（仅拦截新建） | 2026-07-29 |
| 1:1 get-or-create | **强制唯一二人会话：有则复用、无则创建** | 2026-07-29；产品类比微信「一个好友一条会话」；历史若有多条 A–B，复用最近活跃/最近创建之一（design 定），本任务不做合并迁移 |
| 删除会话 / 清空历史 / 列表删除态多端同步 | 待定 | 用户提到类微信能力；是否纳入本父 MVP 见下一题 |

## Product Intent（非默认全做）

- 开聊体验对齐微信会话列表心智：**好友维度一条 1:1**；用户提到「可删历史、可多端同步」——是否进本轮交付见 Open Questions。
- 消息内容本身已在服务端 + 多端拉取/WS，**不等于**「删除会话/已读/列表排序」等列表状态的多端同步产品。

## Open Questions

1. ~~加好友如何找到对方？~~ → 仅邮箱
2. ~~是否双向同意？~~ → 请求→同意
3. ~~`member_emails` 入口？~~ → 完全移除
4. ~~历史非好友会话？~~ → 保留可聊
5. ~~同一对好友是否强制单一 DM？~~ → get-or-create 唯一 1:1
6. 「删除历史 / 会话列表删除 / 该状态多端同步」是否纳入本父任务 MVP？

## Notes

- 父任务保持 `planning`；**先 start / 实现 `friends-relation`**，再 `friends-open-chat`。
- 两子任务均为 complex：start 前各自需要 design + implement（及 jsonl 清单）。
