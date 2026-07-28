# 分组：会话 / 通讯录

**用户可感知含义**：会话列表、创建/加入会话、成员、会话详情。

## 实现状态汇总

implemented / partial：**0**。全部 `not_found`。

## 功能条目

### 会话列表

| 字段 | 内容 |
|------|------|
| status | `not_found` |
| entry | 无页面/路由 |
| code | 无；spec 曾设想 `features/conversation/`（前端）、`internal/service`（后端） |
| deps.tables | 计划可能有 `conversations` / `conversation_members`（见 database guidelines，**无 migrations**） |
| tests | 无 |
| risk | `unknown` |
| newbie_friendly | 列表只读 CRUD 通常较适合新人，**但当前无代码** |
| evidence | 目录与符号均不存在 |

### 创建会话 / 加人

| 字段 | 内容 |
|------|------|
| status | `not_found` |
| entry | 无 |
| code | — |
| deps | — |
| tests | 无 |
| risk | `unknown` |
| evidence | 无 |

### 通讯录 / 好友（若产品需要）

| 字段 | 内容 |
|------|------|
| status | `not_found`（亦未见 spec 强制要求好友模型） |
| entry | 无 |
| evidence | 无代码、无明确产品文档 |
