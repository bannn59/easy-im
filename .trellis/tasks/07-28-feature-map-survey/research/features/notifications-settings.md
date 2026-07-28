# 分组：通知 / 推送 / 设置 / 资料

## 实现状态汇总

implemented / partial：**0**。

## 功能条目

### 离线推送

| 字段 | 内容 |
|------|------|
| status | `planned_only`（`cmd/worker` + `im.push.offline`）/ `not_found` 代码 |
| entry | 用户无直接入口；系统能力 |
| code | 无 |
| deps | 计划 MQ + 推送厂商（未选型落地） |
| tests | 无 |
| risk | 实现后 high |
| newbie_friendly | `false` |
| evidence | 仅 spec |

### 个人资料 / 设置页

| 字段 | 内容 |
|------|------|
| status | `not_found` |
| entry | 无路由/菜单 |
| code | spec 仅示例 `features/settings/` 目录名 |
| tests | 无 |
| risk | `unknown` |
| newbie_friendly | 有页面后通常 true |
| evidence | 无 `frontend/` |

### 管理 / 运维控制台

| 字段 | 内容 |
|------|------|
| status | `not_found` |
| entry | 无 |
| evidence | 无 admin 应用或路由 |
