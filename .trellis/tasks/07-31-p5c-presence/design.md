# Design: P5.c Online Presence (在线/离线圆点)

## Architecture Overview

```
[WS connect] → Hub.Register(userID) ──┐
                                      ├→ 查询好友IDs → Hub.PublishToUsers(好友IDs, presence.changed)
[WS disconnect] → Hub.Unregister(userID) ──┘

GET /v1/friends → FriendService.ListFriends ──> 返回好友列表 ──> handler 查 hub → 每个好友 online 字段
```

- **状态源**：hub 的 `clients` map（在线 = `len(clients[uid]) > 0`），单一真源
- **查询路径**：HTTP（`GET /v1/friends` 初始状态）
- **实时路径**：WS（`presence.changed` 事件）
- **不持久化**：无 DB 变更

## 1. Hub — Presence 查询 + 事件

### 新增方法

```go
// IsOnline reports whether userID has ≥1 live connection.
func (h *Hub) IsOnline(userID string) bool

// OnlineUserIDs returns all user IDs that have ≥1 live connection.
func (h *Hub) OnlineUserIDs() []string
```

- `IsOnline` 加读锁，`return len(h.clients[userID]) > 0`
- `OnlineUserIDs` 加读锁，遍历 map 收集 keys

### Register/Unregister 触发广播

在 `Register` 和 `Unregister` 后调用一个内部 `publishPresence(userID, online)`：

```go
func (h *Hub) publishPresence(userID string, online bool) {
    // 通过 PresenceBroadcaster 回调通知 handler 层
    if h.PresenceBroadcaster != nil {
        h.PresenceBroadcaster(userID, online)
    }
}
```

**关键问题**：hub 是基础设施，不知道"好友"是什么。广播逻辑需要 handler 层（friend repo 查询好友IDs）。

**方案**：hub 暴露 `PresenceBroadcaster func(userID string, online bool)` 回调字段（类似已有的 `FrameHandler`），由 `ws.go`/`router.go` 设置。这样 hub 保持无业务依赖，广播时调用回调。

> **多设备去重**：`Register`/`Unregister` 时判断"状态是否翻转"。第一次连接（0→1）才广播 online；最后一个断开（1→0）才广播 offline。中间连接不重复广播。
> - Register: `wasOnline := len(clients[uid]) > 0`（在插入前检查），插入后如果 `!wasOnline` → 广播 online
> - Unregister: 删除后如果 `len(set) == 0` → 广播 offline

## 2. Friend Service — ListFriendIDs

新增方法返回用户的所有好友 ID：

```go
// FriendStore 新增
ListFriendIDs(ctx context.Context, userID string) ([]string, error)

// FriendService 新增
func (s *FriendService) ListFriendIDs(ctx context.Context, userID string) ([]string, error)
```

**实现**：复用 `ListFriends` 的 SQL，只 SELECT `u.id`（不 hydrate 完整 User）。

## 3. WS Handler — Presence 广播接线

`ws.go` 新增方法：

```go
func (h *WSHandler) broadcastPresence(userID string, online bool) {
    if h.Hub == nil || h.Friends == nil { return }
    ctx := context.Background()
    friendIDs, err := h.Friends.ListFriendIDs(ctx, userID)
    if err != nil || len(friendIDs) == 0 { return }
    payload, _ := json.Marshal(map[string]any{
        "user_id": userID,
        "online":  online,
    })
    h.Hub.PublishToUsers(friendIDs, hub.Event{Type: "presence.changed", Payload: payload})
}
```

`router.go` 接线：
```go
deps.Hub.PresenceBroadcaster = ws.broadcastPresence
```

`WSHandler` 需要新增 `Friends *service.FriendService` 字段。

## 4. Friend Handler — HTTP 查询带 online

`FriendHandler` 需要访问 hub 以计算每个好友的在线状态：

```go
// FriendHandler 新增
Hub *hub.Hub
```

`ListFriends` 改为：
```go
list, err := h.Friends.ListFriends(...)
out := make([]friendWithStatusDTO, 0, len(list))
for _, u := range list {
    out = append(out, friendWithStatusDTO{
        publicUser: toPublicUser(u),
        Online:     h.Hub.IsOnline(u.ID),
    })
}
```

```go
type friendWithStatusDTO struct {
    ID     string `json:"id"`
    Email  string `json:"email"`
    Online bool   `json:"online"`
}
```

> **注意**：不能直接把 `online` 塞进共享的 `publicUser`（auth/me 用），否则 auth 响应也会带 online。用独立 DTO 只用于好友列表。

## 5. Conversation Handler — 会话头在线状态

`GET /v1/conversations/{id}` 和 `GET /v1/conversations` 的 members 需要在线状态。

**决策**：本轮 DM 会话头需要显示对方在线。群聊不做。

方案：`toConversationDTO` 对 members 也输出 `online`。但 `publicUser` 是共享的... 

**简化方案**：会话 DTO 的 members 数组元素用一个带 `online` 的变体。但 conversationDTO.Members 当前是 `[]publicUser`。

```go
// 会话 members 用独立类型
type conversationMemberDTO struct {
    publicUser
    Online bool `json:"online"`
}
```

`toConversationDTO` 遍历 `c.Members` 时填充 `Online: hub.IsOnline(m.ID)`。

> 这要求 `ConversationHandler` 也能访问 hub。当前 `ConversationHandler{Conv}` 需要加 `Hub *hub.Hub`。

**接入点**：handler 层在 `toConversationDTO` 时填充。需要把 hub 传给 ConversationHandler。

## 6. 前端 — 全局 WS 连接重构

### 现状问题

- 前端目前有**两个**独立 WS 连接：`AppShell`（会话列表预览+未读）和 `ConversationRoom`（消息+已读+typing）
- 模块级 `activeWs` 存在竞争：进入房间会覆盖 AppShell 的连接，离开房间 `sendFrame` 可能静默失效
- `FriendsPage` 无 WS 连接，收不到 presence 事件

### 方案：全局单连接 + RealtimeProvider

新建 `RealtimeProvider` context 组件，嵌套在 `SessionProvider` 内（`App.tsx`）：

```
<SessionProvider>
  <RealtimeProvider>     // 新增 — 消费 useSession().token
    <BrowserRouter>...</BrowserRouter>
  </RealtimeProvider>
</SessionProvider>
```

**realtime 模块重构**（`src/realtime/index.ts`）：

```typescript
// 模块级：单一连接 + 订阅者列表
type RealtimeEvent =
  | { type: 'message.created'; payload: Message }
  | { type: 'message.read'; payload: { conversation_id; reader_id; last_read_seq } }
  | { type: 'typing.started'; payload: { conversation_id; user_id } }
  | { type: 'typing.stopped'; payload: { conversation_id; user_id } }
  | { type: 'presence.changed'; payload: { user_id; online } };

// RealtimeProvider 内部：connectRealtime(token, ...) 保持单一连接
// 通过 subscribe(handler) / useRealtime() 分发事件
```

**关键设计**：`RealtimeProvider` 持有单一 `connectRealtime` 连接，内部维护一个 listener 列表。所有组件用 `useRealtime()` hook 订阅感兴趣的事件：

```typescript
function useRealtime(handlers: Partial<RealtimeHandlers>): { status: RealtimeStatus }
```

- 组件挂载时注册 handlers，卸载时注销（连接本身不因组件挂载/卸载而断开）
- `sendFrame` 直接走全局连接的 activeWs（单一真源，无竞争）
- `RealtimeStatus` 通过 context 暴露给需要显示连接状态的组件

**组件改造**：
- `AppShell`：移除自己的 `connectRealtime`，改用 `useRealtime({ onMessageCreated, onMessageRead })` 订阅会话列表更新
- `ConversationRoom`：移除 `connectRealtime`/`sendFrame` 直接调用，改用 `useRealtime({ onMessageCreated, onMessageRead, onTypingStarted, onTypingStopped })`
- `FriendsPage`：新增 `useRealtime({ onPresenceChanged })`，按 `user_id` 更新好友 `online` 状态
- `ConversationRoom` header / FriendsPage 好友行：渲染在线圆点

### 数据模型

`PublicUser` 增加 `online?: boolean`（`api/auth.ts`）：

```typescript
export type PublicUser = { id: string; email: string; online?: boolean };
```

### FriendsPage

- 初始 `GET /v1/friends` 响应已含 `online`（后端 `friendWithStatusDTO`）
- 订阅 `presence.changed` → `setFriends(prev => prev.map(f => f.id === user_id ? {...f, online} : f))`
- 好友行渲染圆点

### ConversationRoom

- `conversation.members` 已含 `online`（初始加载）
- 订阅 `presence.changed` → 更新 peer 的在线状态
- DM header 显示对方圆点

## 7. 事件流总结

### 上线
```
A 建立 WS → Hub.Register → 检测 0→1 翻转 → PresenceBroadcaster(A, true)
  → ws.broadcastPresence → FriendService.ListFriendIDs(A) → [B, C]
  → Hub.PublishToUsers([B, C], presence.changed {user_id: A, online: true})
  → B、C 的全局 WS 收到 → 分发到 FriendsPage / ConversationRoom → 更新 A 的圆点
```

### 下线
```
A 关闭浏览器 → WS 断开 → Hub.Unregister → 检测 1→0 翻转 → 同上广播 online: false
```

### 初始加载
```
B GET /v1/friends → FriendService.ListFriends → 每个好友 Online: hub.IsOnline(id)
  → B 前端渲染圆点
```

## 8. 兼容性 / 回滚

- **DB 无变更**：无 migration
- **API 兼容**：`/v1/friends` 和 `/v1/conversations` 响应新增 `online` 字段，旧前端忽略即可
- **前端重构**：全局单连接是行为等价重构（消息/已读/typing 事件分发路径不变），同时修复 `activeWs` 竞争 bug
- **回滚**：移除 PresenceBroadcaster 赋值 + 前端 presence 处理即可；全局连接重构可单独回滚（保留双连接）
