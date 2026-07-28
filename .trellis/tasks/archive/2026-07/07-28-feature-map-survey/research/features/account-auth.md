# 分组：账号与鉴权

**用户可感知含义**：登录、登出、会话凭证、访问控制入口。

## 实现状态汇总

| 指标 | 值 |
|------|-----|
| implemented | 0 |
| partial | 0 |
| planned_only | 若干（仅 spec） |
| not_found in code | 全部入口 |

## 功能条目

### 登录 / 获取访问凭证

| 字段 | 内容 |
|------|------|
| status | `not_found`（代码）/ 架构上 `planned_only` |
| entry | 无路由、无页面、无 HTTP handler |
| code | — |
| deps | spec 假设：token 校验在 `internal/auth`（**未创建**） |
| tests | 无 |
| risk | `unknown` |
| newbie_friendly | `unknown` |
| evidence | 无 `frontend/`、`backend/`；全库无 auth 相关产品符号 |

### 会话保持 / 刷新令牌

| 字段 | 内容 |
|------|------|
| status | `not_found` |
| entry | 无 |
| code | — |
| deps | — |
| tests | 无 |
| risk | `unknown` |
| evidence | 无实现 |

### WebSocket 升级鉴权

| 字段 | 内容 |
|------|------|
| status | `planned_only`（见 `.trellis/spec/backend/realtime-messaging.md`） |
| entry | 计划：HTTP Upgrade 前鉴权 |
| code | 无 |
| deps | 计划：gateway 进程 |
| tests | 无 |
| risk | 若实现，预期 **high**（安全边界） |
| newbie_friendly | `false`（一旦实现） |
| evidence | 仅 spec，无源码 |

## 备注

本组无任何可点击/可调用的用户入口。
