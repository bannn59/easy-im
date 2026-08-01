# 群成员管理：拉人 / 退群 / 踢人 / 转让群主

**任务**: `08-01-group-members`  
**日期**: 2026-08-01  
**类型**: 复杂任务 — 后端权限 + 前端 UI + WS 实时

## 1. 背景与问题

群聊 MVP（`08-01-group-chat`）已支持建群 + 收发消息 + 成员面板，但**群成员不可变**：
- 群主无法拉人进群（只能建群时定成员）。
- 成员无法退群。
- 无踢人、无群主转让。

需要完整的成员生命周期管理，且成员变化应**实时广播**给群内在线成员（前端成员面板同步）。

## 2. 目标

实现群成员完整管理 + WS 实时广播：

### 权限模型
| 操作 | 谁能做 | 约束 |
|------|--------|------|
| **拉人** | 任意成员 | 被拉者必须是**操作者的好友**；被拉者不能已在群内 |
| **退群** | 自己 | 不能是最后一位成员；群主退群 = 转让或禁止（设计定） |
| **踢人** | 群主 | 不能踢自己；群主不能被踢 |
| **转让群主** | 群主 | 接收者必须是成员；转让后原群主变普通成员 |

### 成功标准（验收）

1. **拉人**：`POST /v1/conversations/{id}/members`（body `{user_ids}`）——成员可拉好友进群；非好友 403；已在群 409。
2. **退群**：`DELETE /v1/conversations/{id}/members/me`——自己退出；最后成员拒绝；群主退群需先转让或禁止（按设计）。
3. **踢人**：`DELETE /v1/conversations/{id}/members/{userID}`——仅群主；非群主 403；踢自己拒绝。
4. **转让群主**：`POST /v1/conversations/{id}/owner`（body `{user_id}`）——仅群主；接收者须为成员。
5. **WS 实时**：成员变化（拉入/退出/踢出/转让）广播 `members.changed` 给群内在线成员，前端成员面板实时更新。
6. **ACL**：被踢/退群后不能再访问群（`GetIfMember` 拒）；被拉入后可见。
7. **前端**：成员面板加「拉人」「退出群聊」「踢人」（群主视图）、「转让群主」（群主视图）操作。
8. **测试全绿**：`go test ./...` + `npm run build`。

### 非目标

- **不做** 群头像/改名。
- **不做** 群公告、禁言、管理员角色（只有群主/普通成员两级）。
- **不做** 被踢/退群的离线通知。

## 3. 约束与边界

- **复用表**：`conversation_members`（增删行）；`conversations.created_by` 存群主。
- **转让群主** = 更新 `conversations.created_by`。
- **成员事件**：复用 `hub.BroadcastToConversation`，新增事件类型 `members.changed`，payload 含 `conversation_id` + 变化描述（action + user_id + 新成员列表或增量）。
- **跨节点**：成员事件跨节点广播需走 Kafka fanout（复用 `im.messages`？或新增——设计定）。**本期至少本地广播**；跨节点后置或复用现有 fanout 机制。
- **好友校验**：拉人需 `friends.AreFriends(operator, target)`；与建群一致。
- **群主唯一**：`created_by` 字段；踢人/转让仅群主。

## 4. 交付物

| # | 交付物 |
|---|--------|
| 1 | repo：成员增删（`AddMembers`/`RemoveMember`）、更新 `created_by`、`GetIfMember` 不变量保持 |
| 2 | service：`AddMembers` / `LeaveGroup` / `KickMember` / `TransferOwner` + 权限校验 |
| 3 | handler：4 个路由（POST members / DELETE me / DELETE {userID} / POST owner） |
| 4 | WS `members.changed` 事件（本地广播；跨节点按 design） |
| 5 | 前端：成员面板操作（拉人/退群/踢人/转让）+ `onMembersChanged` 实时处理 |
| 6 | 测试：service + handler 权限矩阵 + 前端构建 |

## 5. 验收测试

- `cd backend && go test ./...` 全绿。
- `cd frontend && npm run build` 全绿。
- E2E：3 人群 → 成员拉第 4 人（好友）→ 实时广播 → 群主踢人/转让 → 被踢者会话消失。

## 6. 非目标明细（防范围膨胀）

- 不加管理员角色、群公告、禁言。
- 不做被踢/退群离线通知。
- 不做跨节点成员事件（本期本地；如需跨节点复用 fanout，单独评估）。
