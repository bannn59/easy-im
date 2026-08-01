# Design — 群聊：建群 + 收发消息 + 成员面板

**任务**: `08-01-group-chat`  
**日期**: 2026-08-01

---

## 1. 目标形态

```
前端「创建群」对话框
  → POST /v1/conversations/groups  {title, member_ids[]}
      → ConversationService.CreateGroup (校验好友 + 建群 + 多成员)
      → 返回 conversationDTO（含 members + member_count）
  → 前端跳转 /c/:id，WS 实时广播复用现有消息链路
```

**核心**: 现有 `conversations`/`conversation_members` 表 + `repo.Create`（已支持多成员 + title）+ 消息链路（成员驱动广播）已就绪，**只缺 service 建群方法 + handler + 前端入口/面板**。

## 2. 后端

### 2.1 `ConversationService.CreateGroup`

签名: `CreateGroup(ctx, selfUserID string, title *string, memberIDs []string) (domain.Conversation, error)`

步骤:
1. 参数校验: `memberIDs` 非空；总成员 = `memberIDs + selfUserID` ≥ 2（>1 个对方）。去重。
2. 成员上限（防滥用）: 总成员 ≤ 50。
3. 好友校验: 对每个 `memberID`，`friends.AreFriends(selfUserID, memberID)` 必须 true（创建者与每个成员互为好友）。**创建者自己不入校验**。
4. 成员存在性: `users.FindByID(memberID)` 确认存在（好友校验已隐含，但显式更清晰）。
5. 建群: `c := Conversation{Title: title, CreatedBy: selfUserID, ...}`; `convs.Create(ctx, c, allMemberIDs)`（`memberIDs + selfUserID`）。
6. 返回 `convs.GetIfMember(ctx, c.ID, selfUserID)`（带成员列表）。

### 2.2 Handler + 路由

- `POST /v1/conversations/groups`（body `{title, member_ids}`），`require(auth)`。
- 响应 `201` + `conversationDTO`（`toConversationDTO(c, h.Hub)`）。
- 注意路由冲突: 现有 `GET /v1/conversations/{id}` 用 `{id}` 通配。`POST /v1/conversations/groups` 需**精确匹配**（Go 1.22 ServeMux 会优先精确段，`POST /v1/conversations/groups` 与 `POST /v1/conversations/{id}/messages` 不冲突，但与 `GET {id}` 方法不同，安全）。

### 2.3 复用（零改造）

- **消息收发**: `POST /v1/conversations/{id}/messages` 已按成员 ACL + `hub.BroadcastToConversation` 广播——群聊直接可用。
- **WS 实时**: 群内成员在线即收；跨节点由 fanout consumer（`ListMemberIDs` 驱动）覆盖。
- **已读**: `MarkRead` 复用。
- **会话列表/详情**: `List`/`Get` 复用（`member_count`、`members` 已含）。

## 3. 前端

### 3.1 创建群对话框

- 位置: `AppShell` 会话列表工具栏「+」按钮旁，或好友页。
- 内容: 多选好友（`listFriends()`）+ 群名输入框 + 创建按钮。
- 提交: `POST /v1/conversations/groups` → 成功后 `navigate('/c/'+id)`。
- 空群名: 允许（服务端存 NULL title，前端显示「未命名群」——现有逻辑）。

### 3.2 成员面板

- 位置: `ConversationRoom` 标题栏「成员」按钮，点击展开侧栏。
- 内容: `conv.members`（已有，含 online 状态）列表。
- 仅群聊显示（`isGroup`）；单聊不显示或显示对方。

### 3.3 i18n

新增文案: 创建群标题、群名 label、选成员、创建按钮、成员面板标题、未命名群（现有 `chat.groupUntitled`）。

## 4. 数据模型

**不加表**。群 vs 单聊判定 = `member_count > 2`（含创建者）。现有前端 `ConversationRoom.isGroup` 已如此。

## 5. 边界与兼容

- 好友校验严格: 非好友拉群 → `403 not friends`。
- 重复建群: 允许（群聊无唯一约束，不同于 1:1 的 FindDirectBetween）。
- 成员含自己: handler 从 `member_ids` 中去掉 `selfUserID`，避免重复插入。
- 单聊不回归: `OpenDirect` 不动；群聊走新方法。

## 6. 测试

- 后端单测: 建群成功、非好友拒绝、成员过少（<2 总）、含重复 id 去重、成员上限。
- 前端: `npm run build` + 手动 E2E（见 implement.md）。
