# easy-im 功能地图（调研总览）

**任务**: [.trellis/tasks/07-28-feature-map-survey](../)  
**结论一句话**: **当前仓库没有可运行的 IM 产品功能实现**；仅有 Trellis/Agent 脚手架与 bootstrap 架构约定。

| 统计 | 数量 |
|------|------|
| 用户可感知 **implemented** 功能 | **0** |
| **partial** | **0** |
| 仅有架构文档的 **planned_only** 主题 | 若干（见分组页，均无源码） |
| 可导航入口（路由/菜单/命令） | **0** |
| 相关自动化测试 | **0** |

---

## 导航

| 文档 | 说明 |
|------|------|
| [method.md](./method.md) | 扫描范围、空目录清单、复扫命令 |
| [features/account-auth.md](./features/account-auth.md) | 账号与鉴权 |
| [features/conversations.md](./features/conversations.md) | 会话 / 通讯录 |
| [features/messaging.md](./features/messaging.md) | 消息收发与历史 |
| [features/receipts-presence.md](./features/receipts-presence.md) | 回执 / 输入中 / 在线 |
| [features/notifications-settings.md](./features/notifications-settings.md) | 推送 / 设置 / 管理 |
| [notes/gaps-and-next.md](./notes/gaps-and-next.md) | 缺口与脚手架建议 |
| [notes/non-product-surface.md](./notes/non-product-surface.md) | 非产品表面（Trellis 等） |
| [features/_template.md](./features/_template.md) | 单功能字段模板 |

---

## 按用户可感知分组（现状）

| 分组 | 已实现入口 | 代码锚点 | 测试 |
|------|------------|----------|------|
| 账号与鉴权 | 无 | 无 | 无 |
| 会话 / 通讯录 | 无 | 无 | 无 |
| 消息收发与历史 | 无 | 无 | 无 |
| 回执 / 已读 / 输入中 | 无 | 无 | 无 |
| 在线 / 多端 | 无 | 无 | 无 |
| 通知 / 推送 | 无 | 无 | 无 |
| 设置 / 资料 | 无 | 无 | 无 |
| 管理 / 运维 | 无 | 无 | 无 |

每组详情见 `features/*.md`。字段含：功能名、入口、代码、依赖、测试、risk、newbie、evidence。

---

## 高风险 / 高耦合

**现状：无法基于实现代码评级**（无模块）。

若按 bootstrap spec 的**未来**核心路径，下列主题在落地后应默认按 **高风险/高耦合** 管理（**不是**当前代码事实）：

| 主题 | 原因（计划架构） | 文档 |
|------|------------------|------|
| 消息发送 + 扇出 | HTTP/WS + DB + MQ + 多 gateway | `../` → spec `backend/realtime-messaging.md` |
| WS 鉴权与 conn 注册 | 安全边界 + 节点本地状态 | 同上 |
| 在线状态（Redis） | 与 gateway 生命周期耦合；非真相来源易误用 | `database-guidelines.md` |
| 跨层错误码 / 帧版本 | 前后端契约漂移 | `error-handling.md` + frontend `type-safety.md` |

**今日仓库中的「耦合热点」** 实际是工具链（Trellis multi-platform hooks），**不计入产品风险表**。

---

## 适合新人下手

**现状：无产品模块可推荐。**

代码出现后，较适合新人的**预期**切入点（来自 spec 边界，待验证）：

| 候选 | 前提 | 为何可能友好 |
|------|------|--------------|
| HTTP 健康检查 / 版本接口 | `cmd/api` 存在 | 无业务状态 |
| 纯展示设置页 | 设计系统已定 | 少服务依赖 |
| 单一资源只读列表 + 测试 | OpenAPI 已定 | 垂直切片清晰 |
| 消息列表虚拟滚动 UI | mock 数据即可 | 可先不接 WS |

**不建议**新人第一项：gateway 扇出、幂等发送、presence、推送 worker。

---

## 证据原则（本地图遵守）

1. 无文件路径 / 符号 → 不得标 `implemented`。  
2. `.trellis/spec` 只支撑 `planned_only`。  
3. Trellis/Agent 文件不进主功能表。  

详见 [method.md](./method.md)。

---

## 建议的下一步（产品）

见 [notes/gaps-and-next.md](./notes/gaps-and-next.md)。  
最短路径：脚手架 `backend/` + `frontend/` → 再跑一次本调研刷新地图。
