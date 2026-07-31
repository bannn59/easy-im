# PRD — 多节点实时扇出（Kafka-backed fanout）

**任务**: `08-01-p7-fanout-multinode`  
**日期**: 2026-08-01  
**类型**: 复杂任务（父任务）— 架构改造 + 实现

---

## 1. 背景与问题

当前 `internal/hub` 是**单进程内存**广播：

- `MessageService.Send` 存库成功后 → `s.broadcast()` → `rt.PublishToUsers()` 只推给**本节点**的 WS 连接。
- 在线用户**不**走 Kafka（Kafka `im.messages` 只被 `cmd/worker` 消费，用于离线推送）。
- `cmd/worker` 用 `im.presence` 维护在线集合：**在线 → 跳过推送；离线 → 发 Web Push**。

### 缺陷

部署两个 api 实例时（水平扩展）：

| 场景 | 现状 |
|------|------|
| 用户 A 连节点 1、B 连节点 2，A 发消息 | 节点 1 内存广播推不到 B → **B 收不到** |
| B 在线 | worker 判定在线 → **跳过推送** → B 既无实时也无兜底 |
| 多节点 presence | 每个节点只知道自己节点上的连接 → 在线判断不完整 |

**结论**: 实时层无法水平扩展，且多节点时会丢消息（不是优雅降级）。

## 2. 目标

让**每个 api 节点**都能把实时事件送达**任意节点**上的在线用户，实现多实例部署下的在线实时可见。

### 成功标准（验收）

1. **双节点 E2E**：A 连节点 1、B 连节点 2，A 发消息 → B 在**不刷新**下收到 `message.created`。
2. **无重复**：同一节点既有本地直推又消费 Kafka 时，用户不收到两条相同消息（需去重）。
3. **不回退**：单节点（Kafka 不可用）时，消息发送仍秒推、不阻塞；现有测试全绿。
4. **事件覆盖**：跨节点下 `message.created` / `message.edited` / `message.recalled` / `message.read` / `typing.*` / `presence.changed` 都可达（至少核心消息事件，其余按 design 取舍）。
5. **离线推送不回归**：worker 的离线推送逻辑不受影响；在线/离线判定在多节点下语义正确。

### 非目标

- 本任务**不做**：Redis、对象存储、WebSocket 网关独立成进程。
- 不引入新的基础设施依赖（复用现有 Kafka）。
- 不做全站搜索、媒体消息等（见路线图非目标）。

## 3. 约束与边界

- **复用现有 Kafka**（`im.messages` / `im.presence`），不新增 MQ。
- **幂等性基础已具备**：消息有 `client_msg_id`（客户端去重）、DB 唯一约束。
- **发送路径不阻塞**：总线发布失败不影响 HTTP 发送（best-effort，已有 `publishEvent`）。
- **兼容单节点**：无 Kafka 配置时（`KAFKA_BROKERS` 空），api 降级为纯本地广播，功能不退化。
- **worker 独立**：`cmd/worker` 的消费组、presence tracker 保持不变；扇出改造不得破坏离线推送。

## 4. 交付物

1. `internal/mq`：新增扇出事件类型（消息 created/edited/recalled、read、typing 等），复用/扩展 producer。
2. `internal/app`：api 进程内启动一个 **fanout consumer**（独立消费组，如 `easyim-realtime`），消费 `im.messages` 并把事件推给本节点连接的收件人。
3. `internal/hub`：提供「按 conversation 推给本节点成员」的能力 + 去重/来源过滤。
4. 跨节点事件去重（方案见 design：来源标记 + 幂等判断）。
5. E2E 验证脚本/说明（两个 api 实例 + 共享 Postgres/Kafka）。
6. spec 更新：`realtime-messaging.md` 记录多节点扇出架构。

## 5. 验收测试

- `cd backend && go test ./...` 全绿。
- `cd frontend && npm run build` 全绿（前端不强制改动，除非事件形状变化）。
- 双实例手动/脚本验证：见 implement.md 的验证步骤。

## 6. 非目标明细（防范围膨胀）

- 不加消息历史分片、不加跨节点「在线数」聚合 API。
- 不在前端引入多端同步逻辑（这是另一件事，非本次）。
- 不做限流/鉴权重构。
