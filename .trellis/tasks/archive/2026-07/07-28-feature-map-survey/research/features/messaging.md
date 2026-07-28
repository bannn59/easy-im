# 分组：消息收发与历史

**用户可感知含义**：发消息、收消息、历史拉取、重试、撤回（若有）。

## 实现状态汇总

implemented / partial：**0**。

## 功能条目

### 发送消息

| 字段 | 内容 |
|------|------|
| status | `planned_only`（架构）/ `not_found`（代码） |
| entry | 无 Composer UI；无 HTTP/WS send API |
| code | spec：`service.SendMessage`、`client_msg_id` 幂等（`.trellis/spec/backend/realtime-messaging.md` 等） |
| deps | 计划：Postgres 消息表、MQ `im.message.created`、gateway 扇出 |
| tests | 无 |
| risk | 实现后预期 **high**（双通道、幂等、ACL、扇出） |
| newbie_friendly | `false`（核心路径） |
| evidence | 仅 bootstrap spec；无 handler/ws/repo 文件 |

### 历史消息分页

| 字段 | 内容 |
|------|------|
| status | `planned_only` / `not_found` |
| entry | 无 |
| code | spec：keyset/cursor 分页 |
| deps | 计划 Postgres |
| tests | 无 |
| risk | 实现后 **medium**（游标契约前后端对齐） |
| newbie_friendly | 有清晰 API 契约后 **可能** true |
| evidence | 无实现 |

### 实时收消息

| 字段 | 内容 |
|------|------|
| status | `planned_only` / `not_found` |
| entry | 无 WS 客户端模块 |
| code | spec：`frontend/src/realtime/`、`cmd/gateway` |
| deps | Redis 在线、MQ 跨节点 |
| tests | 无 |
| risk | **high**（多节点、重连、去重） |
| newbie_friendly | `false` |
| evidence | 无实现 |

### 消息撤回 / 编辑

| 字段 | 内容 |
|------|------|
| status | `not_found`（spec 仅示例事件名 `im.message.recalled`，无产品确认） |
| entry | 无 |
| evidence | 无代码 |
