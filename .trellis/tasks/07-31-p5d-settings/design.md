# Design: P5.d Settings Page

## Architecture Overview

```
[GET /v1/me] → publicUser + created_at（设置页展示）

[PATCH /v1/me/profile] → AuthService.UpdateDisplayName → UserRepo.UpdateDisplayName
  → 返回更新后 user → 前端 Session 刷新

[POST /v1/me/password] → AuthService.ChangePassword → 验证当前密码 → UserRepo.UpdatePassword
  → 保留当前 token（已确认）
```

- 新增 migration：`users.display_name TEXT NOT NULL DEFAULT ''`
- `publicUser` DTO 增加 `display_name` + `created_at`
- 新路由：`PATCH /v1/me/profile`（改显示名）、`POST /v1/me/password`（改密码）
- **CORS 需加 `PATCH`** 到 `Access-Control-Allow-Methods`

## 1. Migration

**Files:** `backend/migrations/20260731000000_user_display_name.sql`

```sql
-- +goose Up
ALTER TABLE users ADD COLUMN display_name TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE users DROP COLUMN display_name;
```

## 2. Domain & Repo

### domain.User 新增字段

```go
type User struct {
    ID          string
    Email       string
    DisplayName string  // NEW
    CreatedAt   time.Time
    UpdatedAt   time.Time
}
```

### UserRepo 新增方法

```go
// UpdateDisplayName updates display_name and returns the updated user.
func (r *UserRepo) UpdateDisplayName(ctx context.Context, id, displayName string, updatedAt time.Time) (domain.User, error)

// UpdatePassword replaces password_hash.
func (r *UserRepo) UpdatePassword(ctx context.Context, id, hash string, updatedAt time.Time) error
```

SQL 用 `RETURNING id, email, display_name, created_at, updated_at` 避免二次查询。

### 现有查询更新

所有 `SELECT`（Create 插入、FindByEmail、FindByID、ListFriends 的 join、conversation members join）需要包含 `display_name` 列。

## 3. AuthService

### UserStore 接口扩展

```go
type UserStore interface {
    Create(ctx context.Context, rec domain.UserRecord) error
    FindByEmail(ctx context.Context, email string) (domain.UserRecord, error)
    FindByID(ctx context.Context, id string) (domain.User, error)
    FindRecordByID(ctx context.Context, id string) (domain.UserRecord, error)  // NEW — for ChangePassword
    UpdateDisplayName(ctx context.Context, id, displayName string, updatedAt time.Time) (domain.User, error)  // NEW
    UpdatePassword(ctx context.Context, id, hash string, updatedAt time.Time) error  // NEW
}
```

> **决策**：`FindByID` 保持返回无 hash 的 `User`（Me、friend 等现有使用方不受影响）。新增 `FindRecordByID` 专门给 ChangePassword 读 hash。

### 新方法

```go
// UpdateDisplayName sets the user's display name.
func (s *AuthService) UpdateDisplayName(ctx context.Context, userID, displayName string) (domain.User, error)
// - userID 非空校验
// - displayName trim，限长（≤ 64 runes，可为空 = 清除）
// - users.UpdateDisplayName → 返回更新后 user

// ChangePassword verifies the current password, then replaces the hash.
func (s *AuthService) ChangePassword(ctx context.Context, userID, current, newPass string) error
// - userID 非空
// - newPass 走 validatePassword（≥8）
// - users.FindRecordByID → bcrypt.CompareHashAndPassword(current) → 失败 Unauthorized("current password is incorrect")
// - bcrypt.GenerateFromPassword(newPass) → users.UpdatePassword
// - 不触及 token（方案 1）
```

## 4. Handler — AuthHandler 扩展

### publicUser DTO 扩展

```go
type publicUser struct {
    ID          string `json:"id"`
    Email       string `json:"email"`
    DisplayName string `json:"display_name"`
    CreatedAt   string `json:"created_at,omitempty"`
}
```

> 注意：`publicUser` 被 auth、conversation members、friend 复用。给所有公开用户加 `display_name` 是想要的（聊天显示名）。`created_at` 是否全暴露？**决策：只在新 `/v1/me` 响应暴露 created_at**（me 是唯一需要 member-since 的地方），conversation/friend 不带 created_at 以保持轻量。

实现：`publicUser` 加 `display_name`（全场景）；`/v1/me` 单独返回带 `created_at` 的 profile DTO。

```go
type profileDTO struct {
    ID          string `json:"id"`
    Email       string `json:"email"`
    DisplayName string `json:"display_name"`
    CreatedAt   string `json:"created_at"`
}

func (h *AuthHandler) Me(w, r) {
    // ... 现有逻辑
    writeJSON(w, http.StatusOK, profileDTO{...})
}
```

### 新端点

```go
type updateProfileBody struct {
    DisplayName string `json:"display_name"`
}

// PATCH /v1/me/profile
func (h *AuthHandler) UpdateProfile(w, r) {
    // decode → h.Auth.UpdateDisplayName(ctx, UserIDFromContext, body.DisplayName)
    // → writeJSON(profileDTO)
}

type changePasswordBody struct {
    CurrentPassword string `json:"current_password"`
    NewPassword     string `json:"new_password"`
}

// POST /v1/me/password
func (h *AuthHandler) ChangePassword(w, r) {
    // decode → h.Auth.ChangePassword(ctx, UserIDFromContext, current, newPass)
    // → 204/200 { "ok": true }
}
```

### 路由注册 + CORS

```go
mux.Handle("PATCH /v1/me/profile", require(http.HandlerFunc(auth.UpdateProfile)))
mux.Handle("POST /v1/me/password", require(http.HandlerFunc(auth.ChangePassword)))

// withCORS
w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, OPTIONS")
```

## 5. 前端

### 类型

```typescript
// api/auth.ts
export type PublicUser = { id: string; email: string; display_name?: string; online?: boolean };

export type Profile = { id: string; email: string; display_name: string; created_at: string };
```

### 显示名派生工具

`features/chat/types.ts` 的 `shortName` 基础上，新工具：

```typescript
export function displayName(label: string, displayName?: string): string {
  return displayName?.trim() || shortName(label) || label;
}
```

> 命名：优先 display_name → fallback email shortName → fallback 原文。

### Session 刷新

```typescript
type SessionState = {
  token; user; loading;
  login; register; logout;
  setUser: (u: PublicUser | null) => void;  // NEW
  refreshUser: () => Promise<void>;          // NEW — 重新 fetchMe
};
```

### SettingsPage

新 `features/settings/SettingsPage.tsx` + `api/settings.ts`：

- 显示 email + member-since（profile.created_at）
- 显示名表单（复用 FriendsPage 模式：auth guard + form + error + notice + token-passing）
- 改密码表单（current + new + confirm，前端校验两次输入一致）

### 导航

- `App.tsx` 加 `<Route path="/settings" element={<SettingsPage />} />`
- AppShell 侧栏在 signOut 旁加 Settings 链接
- 全部文案走 i18n（新增 `settings` section）

### 聊天界面显示名接入

- `ConversationRoom.memberLabel`：`displayName(email, member.display_name)`
- 群聊发送者名、typing indicator、DM 标题同理
- `AppShell` 侧栏邮箱旁显示 display_name

## 6. 数据流

### 改显示名
```
SettingsPage → PATCH /v1/me/profile {display_name} → AuthService.UpdateDisplayName
  → UserRepo.UpdateDisplayName → 返回 user → profileDTO → 前端 refreshUser()
  → Session.user.display_name 更新 → 所有聊天 UI 用 displayName() 自动反映
```

### 改密码
```
SettingsPage → POST /v1/me/password {current, new} → AuthService.ChangePassword
  → FindByID(带hash) → bcrypt.CompareHashAndPassword(current)
  → GenerateFromPassword(new) → UpdatePassword → 200
  → 前端提示成功；token 不变
```

## 7. 兼容性 / 回滚

- **Migration**：`display_name DEFAULT ''` 向后兼容（旧用户空名 → 前端 fallback 到 email shortName）
- **API**：`publicUser` 加字段是 additive，旧前端忽略；`/v1/me` 从裸对象改为 profileDTO 是 breaking（加了 created_at + display_name 字段），前端同步更新
- **回滚**：移除路由 + 前端页面即可；migration 可 down
