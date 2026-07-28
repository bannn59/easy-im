# 分组：回执 / 已读 / 输入中 / 在线状态

## 实现状态汇总

implemented / partial：**0**。

## 功能条目

### 已读回执

| 字段 | 内容 |
|------|------|
| status | `not_found` |
| entry | 无 |
| code | — |
| deps | — |
| tests | 无 |
| risk | `unknown` |
| evidence | 无符号、无 UI |

### 输入中（typing）

| 字段 | 内容 |
|------|------|
| status | `planned_only`（frontend state/hooks spec 提及）/ `not_found` 代码 |
| entry | 无 |
| risk | 实现后 medium（高频事件、采样） |
| newbie_friendly | 可能 true（若协议已定） |
| evidence | 仅 spec 文字 |

### 在线状态 / 多端

| 字段 | 内容 |
|------|------|
| status | `planned_only`（Redis presence）/ `not_found` 代码 |
| entry | 无 |
| deps | 计划 Redis；非历史真相来源 |
| risk | 实现后 **high**（与 gateway 耦合） |
| newbie_friendly | `false` |
| evidence | `.trellis/spec/backend/realtime-messaging.md`、`database-guidelines.md`；无源码 |
