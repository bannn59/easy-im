# Design — 群成员管理：拉人 / 退群 / 踢人 / 转让群主

**任务**: `08-01-group-members`  
**日期**: 2026-08-01

---

## 1. 目标形态

```
前端成员面板操作
  → POST/DELETE /v1/conversations/{id}/members...
      → ConversationService (权限校验: 群主/成员/好友)
      → repo 增删 conversation_members / 更新 created_by
      → hub.BroadcastToConversation(members, ..., {type:"members.changed", payload})
  → 在线成员 WS 实时收到, 成员面板更新
```

## 2. 权限模型

| 操作 | 角色 | 校验 | 错误 |
|------|------|------|------|
| AddMembers | 任意成员 | 操作者是成员；目标在好友列表；目标未在群 | 403 not member / 403 not friends / 409 already in group |
| LeaveGroup | 自己 | 操作者是成员；不是最后成员 | 403 not member / 409 last member |
| KickMember | 群主 | 操作者是群主；目标是成员且非自己 | 403 not owner / 404 not member / 400 self |
| TransferOwner | 群主 | 操作者是群主；目标在群且非自己 | 403 not owner / 404 not member / 400 self |

**群主退群**：设计为**禁止**（需先转让）。简单、避免群无主。

## 3. Repo 层扩展

新增到 `ConversationStore` 接口 + `ConversationRepo`:

```go
AddMembers(ctx, conversationID string, userIDs []string) error          // INSERT
RemoveMember(ctx, conversationID, userID string) error                  // DELETE
SetOwner(ctx, conversationID, newOwnerID string) error                  // UPDATE created_by
```

`GetIfMember` 已存在（ACL 核心，成员变化后自然生效）。

## 4. Service 层

### 4.1 AddMembers(ctx, conversationID, operatorID string, userIDs []string)
1. 校验 operator 是成员（`GetIfMember`）。
2. 对每个 userID：`users.FindByID` 存在；`friends.AreFriends(operator, target)`；**不在群内**（查现有成员）。
3. `AddMembers` 落库。
4. 广播 `members.changed`（action "added"，含新成员 ids）给**全部成员**（含新增者，让 TA 刷新可见）。

### 4.2 LeaveGroup(ctx, conversationID, operatorID string)
1. operator 是成员。
2. 若 operator == 群主 → 409 "owner must transfer first"。
3. 若成员数 == 1 → 409 "last member cannot leave"。
4. `RemoveMember`。
5. 广播 `members.changed`（action "left"）给剩余成员。

### 4.3 KickMember(ctx, conversationID, operatorID, targetID string)
1. operator == 群主（`convs.Get` 的 `created_by`）。
2. target 是成员且 != operator。
3. `RemoveMember(targetID)`。
4. 广播 `members.changed`（action "kicked"）。

### 4.4 TransferOwner(ctx, conversationID, operatorID, newOwnerID string)
1. operator == 群主。
2. newOwner 是成员且 != operator。
3. `SetOwner(newOwnerID)`。
4. 广播 `members.changed`（action "owner_transferred"）。

## 5. WS 事件：members.changed

payload 结构（新增，前端 `RealtimeHandlers` 加 `onMembersChanged`）:

```json
{
  "type": "members.changed",
  "payload": {
    "conversation_id": "uuid",
    "action": "added|left|kicked|owner_transferred",
    "user_id": "目标用户",           // 新增者/退出者/被踢者/新群主
    "members": ["uuid", ...]         // 变化后的完整成员列表（可选，简化前端刷新）
  }
}
```

- **广播**：`hub.BroadcastToConversation(allMembers, "", event)`——全部成员收。
- **跨节点**：本期**本地广播**。跨节点场景（成员在别的节点）后置——与 `typing.*` 同理（低频、非强一致）。若需跨节点，复用 Kafka fanout consumer 新增事件类型，单独任务。

## 6. Handler + 路由

| 方法+路径 | body | 说明 |
|-----------|------|------|
| `POST /v1/conversations/{id}/members` | `{user_ids: []}` | 拉人 |
| `DELETE /v1/conversations/{id}/members/me` | — | 退群 |
| `DELETE /v1/conversations/{id}/members/{userID}` | — | 踢人 |
| `POST /v1/conversations/{id}/owner` | `{user_id}` | 转让群主 |

Go 1.22 ServeMux 路由兼容性：
- `DELETE /v1/conversations/{id}/members/me` 与 `DELETE /v1/conversations/{id}/members/{userID}` 冲突？——**`me` 是精确段，`{userID}` 是通配段**。ServeMux 精确段优先，`me` 匹配到退群，其他 UUID 匹配到踢人。**不冲突**（验证）。

## 7. 前端

- `src/api/conversations.ts`：`addMembers` / `leaveGroup` / `kickMember` / `transferOwner`。
- `ConversationRoom` 成员面板：群主视图显示「踢人」「转让群主」；所有成员显示「退出群聊」；「拉人」入口（对话框复用 CreateGroupDialog 的多选逻辑，抽成通用 MemberPicker）。
- `src/realtime/index.tsx`：`RealtimeHandlers` 加 `onMembersChanged`；`ConversationRoom`/`AppShell` 消费——更新 `conv.members`、刷新会话列表。

## 8. 测试

- service：权限矩阵（非群主踢人 403、群主踢自己 400、非好友拉人 403、最后成员退群 409、群主退群 409、转让非成员 404）。
- handler：路由 + 状态码。
- 前端：`npm run build` + 手动 E2E。

## 9. 边界

- 转让群主后，原群主变普通成员（可被新群主踢）。
- 被踢/退群者的会话在下次 `List` 时消失（`ListForUser` 按 membership 过滤，天然正确）。
- `members.changed` payload 的 `members` 完整列表：拉人/踢人后前端可直接 `setConv({...conv, members})`，避免再拉详情。
