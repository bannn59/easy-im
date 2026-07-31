# Implement: P5.d Settings Page

## Implementation Order

后端（migration → domain/repo → service → handler → CORS），前端（类型 → Session → SettingsPage → 导航 → 显示名接入）。

---

### Step 1: Migration

**Files:** `backend/migrations/20260731000000_user_display_name.sql`

- [x] `+goose Up`: `ALTER TABLE users ADD COLUMN display_name TEXT NOT NULL DEFAULT ''`
- [x] `+goose Down`: `ALTER TABLE users DROP COLUMN display_name`

**Verify:** `go build ./...`

### Step 2: Domain & Repo

**Files:** `backend/internal/domain/user.go`, `backend/internal/repo/user_repo.go`

- [x] `domain.User` 加 `DisplayName string`
- [x] `Create` 插入 `display_name`
- [x] `FindByEmail` SELECT `display_name`
- [x] `FindByID` SELECT `display_name`
- [x] `FindIDsByEmails` 保持（只取 id/email）
- [x] 新增 `FindRecordByID(ctx, id) (domain.UserRecord, error)` — 含 hash + display_name
- [x] 新增 `UpdateDisplayName(ctx, id, displayName, updatedAt) (domain.User, error)` — RETURNING
- [x] 新增 `UpdatePassword(ctx, id, hash, updatedAt) error`

**Verify:** `go build ./...` + 测试通过

### Step 3: 现有 SELECT 更新（display_name 列）

**Files:** `backend/internal/repo/friend_repo.go`, `backend/internal/repo/conversation_repo.go`（若有 user join）

- [x] `ListFriends` join users 时 SELECT `display_name`
- [x] `ListIncomingPending` join users 时 SELECT `display_name`
- [x] conversation members attach 的 user join 含 `display_name`

**Verify:** `go build ./...`

### Step 4: AuthService

**Files:** `backend/internal/service/auth_service.go`

- [x] `UserStore` 接口加 `FindRecordByID`、`UpdateDisplayName`、`UpdatePassword`
- [x] `UpdateDisplayName(ctx, userID, displayName) (domain.User, error)`：
  - userID 非空；displayName trim，≤64 runes（可为空）
  - `users.UpdateDisplayName`
- [x] `ChangePassword(ctx, userID, current, newPass) error`：
  - userID 非空；newPass 走 `validatePassword`
  - `users.FindRecordByID` → bcrypt.CompareHashAndPassword(current) → 失败 `Unauthorized("current password is incorrect")`
  - bcrypt.GenerateFromPassword(newPass) → `users.UpdatePassword`

**Verify:** `go build ./...` + 新增单元测试

### Step 5: Handler

**Files:** `backend/internal/handler/auth.go`, `backend/internal/handler/router.go`

- [x] `publicUser` 加 `DisplayName string \`json:"display_name"\``
- [x] `toPublicUser` 填 `DisplayName`
- [x] 新增 `profileDTO { id, email, display_name, created_at }`
- [x] `Me` 返回 `profileDTO`（含 created_at）
- [x] 新增 `UpdateProfile` handler（PATCH `/v1/me/profile`）
- [x] 新增 `ChangePassword` handler（POST `/v1/me/password`）
- [x] router.go 注册两个新路由（require 保护）
- [x] withCORS 加 `PATCH` 到 `Access-Control-Allow-Methods`

**Verify:** `go build ./...` + handler 测试

### Step 6: 后端测试

**Files:** `backend/internal/service/auth_service_test.go`

- [x] `memUsers` 加 `FindRecordByID`/`UpdateDisplayName`/`UpdatePassword`
- [x] UpdateDisplayName：成功改名、清空名、超长拒绝
- [x] ChangePassword：正确改密、当前密码错拒绝、新密码太短拒绝、改后旧密码登录失败新密码成功

**Verify:** `go test ./...` 全绿

### Step 7: 前端 — 类型 + Session

**Files:** `frontend/src/api/auth.ts`, `frontend/src/app/Session.tsx`

- [x] `PublicUser` 加 `display_name?: string`
- [x] 新增 `Profile` 类型 `{ id, email, display_name, created_at }`
- [x] `fetchMe` 类型改为 Profile
- [x] Session 加 `setUser` 和 `refreshUser`

**Verify:** `tsc --noEmit`

### Step 8: 前端 — 显示名工具 + API

**Files:** `frontend/src/features/chat/types.ts`, `frontend/src/api/settings.ts`(新)

- [x] `displayName(emailOrLabel, displayName?)` 工具：displayName → shortName → label
- [x] `api/settings.ts`：`updateProfile(token, displayName)`、`changePassword(token, current, newPass)`

**Verify:** `tsc --noEmit`

### Step 9: 前端 — SettingsPage + 导航

**Files:** `frontend/src/features/settings/SettingsPage.tsx`(新), `frontend/src/app/App.tsx`, `frontend/src/app/AppShell.tsx`, `frontend/src/i18n/locales/en.json`, `frontend/src/i18n/locales/zh-CN.json`

- [x] `SettingsPage`：profile 展示（email + member-since）+ 显示名表单 + 改密码表单（复用 FriendsPage 模式）
- [x] `App.tsx` 加 `/settings` 路由
- [x] `AppShell` 侧栏加 Settings 链接
- [x] i18n：新增 `settings` section（en + zh-CN）

**Verify:** `tsc --noEmit`

### Step 10: 前端 — 显示名接入聊天 UI

**Files:** `frontend/src/features/chat/ConversationRoom.tsx`, `frontend/src/app/AppShell.tsx`

- [x] `ConversationRoom.memberLabel` 用 `displayName(email, member.display_name)`
- [x] 群聊发送者名、typing indicator、DM 标题优先显示 display_name
- [x] `AppShell` 侧栏邮箱旁显示 display_name

**Verify:** `tsc --noEmit` + 手动验证

---

## Risky Files / Rollback Points

| File | Risk | Rollback |
|------|------|----------|
| `user_repo.go` | 所有 SELECT 加 display_name 列，改漏会导致 scan 错位 | 编译器保护（scan 数量不匹配会报错） |
| `auth.go` | `/v1/me` 从裸对象改 profileDTO（breaking） | 前端同步改 fetchMe 类型 |
| `auth_service.go` | ChangePassword 流程错误可能锁死账号 | 单元测试覆盖 |

## Validation Commands

```bash
# Backend
go build ./...
go test ./...

# Frontend
npx tsc --noEmit

# Manual
# 1. 登录 → 打开 /settings → 看到 email + 注册时间
# 2. 改显示名 → 保存 → DM 标题/群聊名字更新
# 3. 改密码 → 登出 → 旧密码登录失败 → 新密码登录成功
```
