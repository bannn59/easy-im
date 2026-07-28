# easy-im 功能开发路线图

**任务**: `07-28-feature-dev-roadmap`  
**日期**: 2026-07-28  
**性质**: 规划文档（无产品代码）

**依据**:

- 功能地图（归档）: `.trellis/tasks/archive/2026-07/07-28-feature-map-survey/research/`
- 脚手架: `backend/cmd/api`（`GET /healthz`）、`frontend`（`/`、`/health`）
- Spec: `.trellis/spec/backend/*`、`frontend/*`、`guides/*`

---

## 1. 原则

1. **垂直切片**：每个阶段结束都有用户能感知的结果（或明确的 API + 最小 UI）。
2. **先 HTTP 后 WS**：消息先落库并可刷新可见，再上实时；多节点 MQ 更后。
3. **契约先于功能膨胀**：错误码、`client_msg_id`、会话内 `seq` 在进消息期定住。
4. **高风险后置**：多 gateway 扇出、presence 当真相、离线推送、双写。
5. **地图可更新**：每完成一个用户可感知切片，重跑或补丁功能地图 research。
6. **Spec 不是实现**：bootstrap 约定只指导怎么做，不计入「已上线」。

```text
M0 脚手架 ✅
  → P0 地基 → P1 身份 → P2 会话 → P3 消息(HTTP) → P4 实时(WS)
       → P5 体验 → P6 规模/运维
```

---

## 2. 功能地图校准（脚手架之后）

| 地图分组 | 调研时 | **当前** | 证据 |
|----------|--------|----------|------|
| 基建 / 健康（地图外补丁） | 无 | **partial** | `GET /healthz`；FE `/`、`/health` |
| 账号与鉴权 | not_found | not_found | 无 auth 路由/表 |
| 会话 / 通讯录 | not_found | not_found | 无 |
| 消息收发与历史 | not_found | not_found | 无 |
| 回执 / 输入中 | not_found | not_found | 无 |
| 在线 / 多端 | not_found | not_found | 无 gateway |
| 通知 / 推送 | not_found | not_found | 无 worker |
| 设置 / 资料 | not_found | not_found | 无 |
| 管理 / 运维 | not_found | not_found | 无 |

归档地图仍写「0 实现 / 无 backend」——**部分过时**；以本表 + 代码为准，完整复扫另开任务。

---

## 3. 阶段详述

### P0 — 开发地基

| 项 | 用户感知 | 交付 | 风险 | 新人 |
|----|----------|------|------|------|
| Postgres + 迁移工具链 | 无 | `migrations/` 可跑；文档或 compose | 低 | ✅ |
| 统一错误 JSON + `request_id` | 失败可理解 | middleware / apperr | 低 | ✅ |
| 配置扩展（`DATABASE_URL` 等） | 无 | `internal/config` | 低 | ✅ |
| （可选）`packages/contracts` 挂点 | 无 | 目录或 OpenAPI 雏形 | 低 | ✅ |

**出口**: 本地 DB 可连（或文档清晰）；API 错误形状固定；`/healthz` 仍绿。

**可与 P1 前半并行。**

---

### P1 — 账号与鉴权  
（地图: `features/account-auth.md`）

| 切片 | 用户能做什么 | 后端 | 前端 |
|------|--------------|------|------|
| **P1.1** 注册/登录 | 注册、登录拿 token | `users`；`POST /auth/register\|login` | `/login` `/register` |
| **P1.2** 会话保持 | 刷新仍登录 | Bearer；`GET /me` | Session + 路由守卫 |
| **P1.3** WS 票 | （可延到 P4） | short-lived ticket | — |

**不做**: OAuth、多租户、复杂 RBAC。  
**出口**: 登录后进入主框（侧栏占位 + 当前用户）。  
**风险**: 中（安全）。**新人**: UI 可跟；密码/JWT 建议熟手把关。

---

### P2 — 会话  
（地图: `features/conversations.md`）

| 切片 | 用户能做什么 | 后端 | 前端 |
|------|--------------|------|------|
| **P2.1** 创建会话 | 建 1:1 或最小群 | `conversations` + `members`；`POST /conversations` | 创建入口 |
| **P2.2** 列表 | 看见会话列表 | `GET /conversations` | 侧栏 |
| **P2.3** 打开会话 | 进入空聊天页 | 成员 ACL | `/c/:id` |

**出口**: 两用户（两浏览器）进入同一会话。  
**风险**: 低–中（ACL 必须在 service）。**新人**: 列表 UI ✅。

---

### P3 — 消息 · HTTP 优先 ⭐  
（地图: `features/messaging.md`）

| 切片 | 用户能做什么 | 关键设计 | 前端 |
|------|--------------|----------|------|
| **P3.1** 发送+落库 | 发出一条消息 | `messages`；**server seq**；**`client_msg_id` 幂等**；ACL | Composer + 乐观 UI |
| **P3.2** 历史 | 看得到历史 | keyset cursor | 消息列表 |
| **P3.3** 对方可见 | B **刷新**后看到 A | 同上 | 临时刷新/短轮询可接受 |

**出口**: 无 WS 时 A 发、B 刷新可见；重复提交不双泡。  
**风险**: 高（契约）但可控。**新人**: 纯 UI/mock ✅；接线建议熟手。

这是第一个「像 IM」的产品里程碑。

---

### P4 — 实时  
（地图: messaging + spec `realtime-messaging.md`）

| 切片 | 用户能做什么 | 后端 | 前端 |
|------|--------------|------|------|
| **P4.1** 单机 gateway | 在线秒收 | `cmd/gateway`；鉴权；conn 注册；推送事件 | `realtime/` 真连接；按 id upsert |
| **P4.2** 共用 use-case | HTTP/WS 发送一致 | **同一** `SendMessage` | 去掉轮询 |
| **P4.3** 跨节点（可选） | 多实例仍达 | NATS/Kafka 扇出 | 无感 |

**出口**: 双开页面，A 发 B 不刷新收到。  
**风险**: **高**。**新人**: ❌ 勿挂 P4 主路径。

---

### P5 — 体验增强  
（地图: receipts-presence / settings）

建议 **P3 之后** 按需插入：

| 优先级 | 功能 | 风险 | 新人 |
|--------|------|------|------|
| P5.a | 已读/回执 | 中 | 部分 |
| P5.b | 输入中 | 中（高频） | 部分 |
| P5.c | 在线（Redis） | **高**（勿当 ACL/历史真相） | ❌ |
| P5.d | 资料/设置页 | 低 | ✅ |
| P5.e | 撤回/编辑 | 中（事件版本） | 部分 |

---

### P6 — 规模与运维  
（地图: notifications / 管理）

| 项 | 说明 |
|----|------|
| 离线推送 `cmd/worker` | P4 稳定后；高风险 |
| 管理后台 | 可长期不做 |
| 可观测性 | metrics + 已有 logging 字段约定 |
| 生产 CORS/密钥 | 上环境前收紧（脚手架现为 dev `*`） |

---

## 4. 建议 Trellis 任务切分

| 序 | 建议标题 | 对应 | 验收一句话 |
|----|----------|------|------------|
| T1 | 本地 Postgres + 迁移 + 错误中间件 | P0 | DB 可迁；统一 error JSON |
| T2 | 注册登录 + `/me` + 登录页 | P1 | 登录进主框 |
| T3 | 会话创建/列表 + ACL | P2 | 两人同会话 |
| T4 | HTTP 发消息 + 历史 + client_msg_id | P3 | 刷新可见且幂等 |
| T5 | 单机 WS gateway + realtime 客户端 | P4 | 在线秒收 |
| T6 | 刷新功能地图 research | 文档 | implemented > 0 有证据 |

依赖写在各子任务 PRD 中（T3 依赖 T2，T4 依赖 T3，T5 依赖 T4）；**不要**只靠 parent/child 暗示顺序。

---

## 5. 里程碑（产品视角）

| 里程碑 | 定义 | 状态 |
|--------|------|------|
| **M0** | 前后端可启动 + health | ✅ 已完成 |
| **M1** | 能登录 | 未做 |
| **M2** | 能建会话并打开 | 未做 |
| **M3** | 能发消息且对方刷新可见 | 未做 |
| **M4** | 在线实时可见（WS） | 未做 |
| **M5** | 已读或在线状态之一 | 未做 |

---

## 6. 明确非目标（避免范围膨胀）

第一期默认 **不上**：

- 音视频、消息全站搜索、复杂好友图谱  
- OAuth / SSO、多租户  
- 把 Redis 当消息历史  
- 为「目录齐全」而建空的 `cmd/gateway`/`worker` 伪实现  
- 离线推送、管理后台作为 M3 前置  

---

## 7. 风险与新人（汇总）

**高风险 / 高耦合（后置）**

- 消息发送 + 多节点扇出（HTTP+WS+DB+MQ）  
- WS 鉴权与 conn 注册  
- Presence 误用为 ACL 或历史  
- 跨层帧/错误码漂移  

**相对适合新人（有前提）**

- P0 迁移/错误中间件（有模板时）  
- P2 会话列表 UI  
- P5.d 设置页  
- P3 消息气泡纯展示（mock）  
- 已完成的 health 类改动  

---

## 8. 下一步默认开做

**推荐下一个实现任务**: **T1 + T2 可拆两任务，或「P0 最小 + P1.1 注册登录」合并为鉴权闭环（若希望更快见到登录页，P0 只做迁移+错误码与 P1.1 同 PR 亦可）。**

更稳默认：

1. **T1** Postgres + migrations + error middleware  
2. **T2** 注册/登录 + `/me` + 前端登录  

达成 **M1** 后再开 T3。

---

## 9. 复盘触发

出现以下情况应更新本路线图或重跑功能地图：

- 技术选型变更（例如不上 Postgres）  
- 跳过 P3 直接上 WS 的决策（需显式记录风险）  
- M3/M4 完成后（勾里程碑、刷新地图）  
