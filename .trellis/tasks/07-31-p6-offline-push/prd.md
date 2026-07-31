# P6 Offline push (Web Push PWA)

## Goal

让用户在**浏览器关闭 / 标签页后台**时也能收到新消息系统通知：采用 **Web Push（PWA）** 方案，后端存储订阅、在目标用户离线时发送 push，前端 Service Worker 接收并展示通知。

## Background

### 已确认事实（代码侦察）

- **消息闭环**：`MessageService.Send`（HTTP）落库后经 `broadcast` → `rt.PublishToUsers`（`internal/hub` 内存扇出）推 `message.created` 给会话全部成员。`hub.IsOnline(userID)` 可判断用户是否有活跃 WS 连接。
- **进程形态**：单进程 `cmd/api` 同时服务 HTTP + dev WS hub。`cmd/worker` 仍是 spec 中的 "Planned"，无 MQ。
- **认证**：HttpOnly cookie 会话（`easyim_session`），WS 从 cookie 读 JWT。推送订阅归属需要用户身份 → 用受保护 API 完成注册。
- **前端**：React + Vite + TypeScript。**无 Service Worker / manifest / PWA 基础设施**。有 `frontend/src/api/*`（auth/client/http 等）与 `app/Session.tsx`（会话状态）。
- **库选型**：`SherClockHolmes/webpush` 在 Go module proxy 已不可用（404）。可用替代 **`github.com/wuc656/webpush-go`**（fork，支持 VAPID + RFC 8291 aes128gcm 加密，v1.4.8，proxy 200）。

### 用户已确认

- 方案形态：**Web Push（PWA）**（非 App 内离线态、非 APNs/FCM 移动端）。

## Requirements

- **R1 推送订阅注册**：后端存储用户的 Web Push 订阅（endpoint + keys），受登录保护。
- **R2 离线判定 + 触发**：新消息发送时，对**无活跃连接**的会话成员发送 Web Push。
- **R3 通知内容**：标题为发送者，正文为消息预览，点击通知打开对应会话。
- **R4 前端接收**：Service Worker 接收 push、展示通知；页面内（前台）不重复打扰。
- **R5 失效订阅清理**：push 服务返回 410 Gone / 404 时删除失效订阅。
- **R6 配置**：VAPID 密钥走 env；开发环境（localhost / 明文 HTTP）可用。

## Acceptance Criteria

- [ ] 用户在设置页开启「推送通知」后，请求 Notification 权限并注册订阅，订阅持久化到 DB。
- [ ] 目标用户所有连接离线时收到系统通知（标题=发送者、正文=消息预览）。
- [ ] 用户在线时不触发系统通知（避免前台打扰）。
- [ ] 点击通知打开对应会话。
- [ ] 失效订阅被自动清理。
- [ ] 后端/前端测试通过；`go vet` / `tsc` / lint 干净。

## Out of Scope

- 移动端 APNs / FCM 推送。
- 推送撤回 / 通知 badge 计数同步。
- 多节点 MQ 扇出（当前单进程）。
- 管理后台 / 可观测性（P6 其他项）。

## Key Decisions

- **触发架构（已确认）**：**独立 `cmd/worker` + MQ**。消息发送时发布事件到 MQ，worker 进程订阅消费、判断目标用户离线后发 push。引入 MQ 基础设施（当前仓库无）。
- **MQ 选型（已确认）**：**Kafka**。依据：生产级 IM 的主流做法（OpenIM 用 Kafka 扇出 + toOfflinePush 离线通道）；通知系统设计以 Kafka 解耦/缓冲/重放/分区。消息按会话分区保证有序，offset 提交语义支持 at-least-once 重放。
- **Kafka Go 客户端（已确认）**：**`github.com/twmb/franz-go`**（纯 Go、无 cgo、API 稳定）。
- **订阅时机（已确认）**：**设置页开关**。登录后在 `/settings` 页新增「推送通知」开关，用户主动开启时请求 Notification 权限并注册订阅（符合浏览器需用户手势的约束，尊重用户选择）。
- **通知聚合（已确认）**：**会话聚合**。同一会话离线期多条新消息合并为一条「N 条新消息」系统通知（含最新消息预览）。实现采用后端聚合窗口（worker 对同会话消息在时间窗内合并发送），带 `tag: conversationID` 让通知替换而非轰炸。

## Risks

- **webpush-go 采用度低**：`wuc656/webpush-go` 是 SherClockHolmes 的 fork，功能完整但 `Imported by: 0`。VAPID/加密按标准实现（RFC 8291），vendor 后可降低供应风险。
- **Kafka 运维成本**：本项目首个消息中间件，需新增 broker 到 docker-compose；单机开发环境占用内存明显。
- **离线判定依赖 presence 事件流**：worker 从 Kafka 学到的在线状态有秒级延迟；极短窗口内的「离线用户刚上线即发 push」可能重复打扰（可接受）。
