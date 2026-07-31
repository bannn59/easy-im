# P5.d Settings Page

## Goal

A dedicated settings page where users view their profile (email, member-since) and manage account credentials: display name and password.

## Background

P5.d was tagged **Low risk / newcomer-friendly**. Currently the only account surface is the email shown in the AppShell sidebar. The user model is minimal (`ID`, `Email`, `CreatedAt`, `UpdatedAt`) with no display name or password-change flow.

### Current technical foundation (confirmed by research)

- `User` domain: `ID`, `Email`, `CreatedAt`, `UpdatedAt`; no display name
- `UserRepo`: Create/FindByEmail/FindByID/FindIDsByEmails — no update methods
- `AuthService`: Register/Login/Me/ParseAccessToken; bcrypt already used
- Frontend derives all names from email (`shortName` takes local part before `@`)
- Session context holds `{ token, user, loading, login, register, logout }` — no refresh/setUser
- Form patterns: AuthPage (form/error/disabled), FriendsPage (notice + auth guard)
- CORS currently allows only GET/POST/OPTIONS

## Scope (confirmed)

Complete settings page: **read-only profile + editable display name + change password**.

## Requirements

### R1 — Profile 展示

- 设置页显示当前用户 email 和注册时间（member since）
- 需要后端暴露 `created_at`（当前 `publicUser` DTO 不含时间戳）

### R2 — 显示名 (Display Name)

- 用户可编辑 display name
- 聊天界面（DM 标题、群聊发送者名、typing indicator）优先显示 display name，fallback 到 email shortName
- 好友列表、会话成员同理
- 前端 Session 在更新后刷新 user

### R3 — 改密码

- 用户需输入当前密码 + 新密码（+确认）
- 后端验证当前密码，替换为新密码 hash
- **Token 策略（已确认）**：改密码后保留当前 token；不注销任何设备（token 版本机制推迟到 P6）

### R4 — 路由与导航

- 新路由 `/settings`
- Header 或 AppShell 侧栏提供入口

## Acceptance Criteria

- [ ] 设置页显示 email + 注册时间
- [ ] 用户可修改 display name，保存后：
  - 设置页立即显示新名字
  - DM 标题、群聊发送者名、typing indicator 使用新名字
  - 刷新页面后名字持久保留
- [ ] 改密码流程：
  - 当前密码错误 → 报错，不修改
  - 新密码合法 → 成功；下次登录必须用新密码，旧密码失效
  - 成功后当前会话保持登录
- [ ] 所有 UI 文案走 i18n（en + zh-CN）
- [ ] 未登录访问 `/settings` 重定向到 `/login`
- [ ] CORS 支持新增的写方法（若引入 PATCH/PUT）

## Out of Scope

- 头像上传（无存储基础设施）
- Token 版本 / 踢下线其他设备（P6）
- 邮箱修改 / 邮箱验证
- 通知偏好、隐私设置
- 账号删除
- 主题 / 外观设置

## Open Questions

_(none — all blocking decisions resolved)_
