# Implement: P5.c Online Presence (在线/离线圆点)

## Implementation Order

后端 bottom-up（hub → friend → ws → handler），然后前端（realtime 重构 → presence UI）。

---

### Step 1: Hub — Presence 查询 + 翻转广播

**Files:** `backend/internal/hub/hub.go`

- [x] 新增 `IsOnline(userID string) bool`
- [x] 新增 `OnlineUserIDs() []string`
- [x] 新增 `PresenceBroadcaster func(userID string, online bool)` 字段
- [x] `Register`: 0→1 翻转广播 online
- [x] `Unregister`: 1→0 翻转广播 offline
- [x] `publishPresence` 私有方法（nil-safe）

**Verify:** `go build` + hub 测试通过 ✅

### Step 2: Friend Store — ListFriendIDs

**Files:** `backend/internal/repo/friend_repo.go`, `backend/internal/service/friend_service.go`

- [x] `FriendRepo.ListFriendIDs`（只 SELECT id）
- [x] `FriendStore` 接口新增 `ListFriendIDs`
- [x] `FriendService.ListFriendIDs` 转发
- [x] 测试 mock `memFriendStore.ListFriendIDs`

**Verify:** `go test` 通过 ✅

### Step 3: WS Handler — Presence 广播

**Files:** `backend/internal/handler/ws.go`, `backend/internal/handler/router.go`

- [x] `WSHandler` 新增 `Friends` 字段
- [x] `broadcastPresence` 方法（ListFriendIDs → PublishToUsers）
- [x] `router.go`：`PresenceBroadcaster = ws.broadcastPresence`
- [x] `api.go` 已传 Friends

**Verify:** `go build` 通过 ✅

### Step 4: Friend Handler — online 字段

**Files:** `backend/internal/handler/friend.go`, `backend/internal/handler/router.go`

- [x] `FriendHandler` 新增 `Hub`
- [x] `friendWithStatusDTO { id, email, online }`
- [x] `ListFriends` 填充 online
- [x] `router.go` 传 Hub

**Verify:** `go build` + 测试通过 ✅

### Step 5: Conversation Handler — members online

**Files:** `backend/internal/handler/conversation.go`, `backend/internal/handler/router.go`

- [x] `ConversationHandler` 新增 `Hub`
- [x] `conversationMemberDTO { id, email, online }`
- [x] `toConversationDTO(c, hub)` 签名变更 + 所有调用方更新
- [x] `router.go` 传 Hub

**Verify:** `go build` + 测试通过 ✅

### Step 6: 后端测试

**Files:** `backend/internal/hub/presence_test.go`(新), `backend/internal/service/friend_service_test.go`

- [x] hub 测试：翻转语义、IsOnline、OnlineUserIDs（真实 WS 连接）
- [x] friend service 测试：ListFriendIDs 断言

**Verify:** `go test ./...` 全绿 ✅

### Step 7: 前端 — Realtime 全局单连接重构

**Files:** `frontend/src/realtime/index.ts`, `frontend/src/app/App.tsx`, `frontend/src/app/AppShell.tsx`, `frontend/src/features/chat/ConversationRoom.tsx`

- [x] `RealtimeProvider` context 组件（连接生命周期跟 token）
- [x] 单例连接 + subscriber 集合 + `useRealtime` hook
- [x] `sendFrame` 走全局连接
- [x] 新增 `onPresenceChanged` handler + `presence.changed` case
- [x] `App.tsx` 嵌套 `RealtimeProvider`
- [x] `AppShell` 改用 `useRealtime`
- [x] `ConversationRoom` 改用 `useRealtime`（15s 轮询独立）

**Verify:** `tsc --noEmit` 通过 ✅

### Step 8: 前端 — Presence UI

**Files:** `frontend/src/api/auth.ts`, `frontend/src/features/friends/FriendsPage.tsx`, `frontend/src/features/chat/ConversationRoom.tsx`, `frontend/src/styles/index.css`

- [x] `PublicUser.online?`
- [x] `FriendsPage`：presence 订阅 + 好友行圆点
- [x] `ConversationRoom`：DM header 圆点 + presence override
- [x] CSS：presence-dot（在线实心/离线空心）

**Verify:** `tsc --noEmit` 通过 ✅

---

## Risky Files / Rollback Points

| File | Risk | Rollback |
|------|------|----------|
| `hub/hub.go` | Register/Unregister 触发广播，可能影响现有连接管理 | 移除 `PresenceBroadcaster` 赋值 |
| `realtime/index.ts` | 全局单连接重构最大风险点 | 保留旧 `connectRealtime` 导出，组件回退 |
| `conversation.go` | members DTO 变更可能影响现有前端 | 只加字段，不改既有字段 |

## Validation Commands

```bash
# Backend
go build ./...
go test ./...

# Frontend
npx tsc --noEmit

# Manual presence test
# 1. A、B 两个浏览器登录
# 2. B 打开 /friends → 看到 A 在线圆点
# 3. A 关闭页面 → B 的好友列表 A 变离线（无需刷新）
# 4. B 打开 A 的 DM 会话 → header 显示 A 在线状态
```
