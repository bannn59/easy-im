# 可观测性metrics与并发压测

**任务**: `08-01-obs-loadtest`  
**日期**: 2026-08-01  
**类型**: 复杂任务（父任务）— 开发期轻量可观测性 + 并发压测基线

## 1. 背景与问题

easy-im 已实现 M0–M5 全部里程碑 + 超额功能（多节点 Kafka 扇出、离线推送、PWA、i18n）。当前代码：

- **有** `log/slog` 结构化日志 + `X-Request-ID` 关联（api/worker 一致），有 `/healthz` + `/readyz`。
- **无** 任何 metrics 端点（无 Prometheus、无 `/metrics`）。
- **无** 并发压测基线：不知道单机吞吐，也不知道水平扩展到 2–3 节点（经 LB）能提升多少。

**结论**: 缺少「并发能力」的可量化基线。后续优化、容量规划都无从谈起。

## 2. 目标

1. **开发期轻量可观测性**：为 HTTP / WS / 消息 / Kafka / 推送路径埋点，暴露 Prometheus `/metrics`。轻量、可开关，不引入 OTel。
2. **并发压测基线**：用 `wrk` 压测单机 vs 水平扩展 2–3 节点（经负载均衡器）的并发吞吐，产出可复现的对比报告。

### 成功标准（验收）

1. [x] **metrics 可用**：`GET /metrics` 返回 Prometheus 文本格式；覆盖 HTTP 请求延迟/计数、WS 在线连接数、消息吞吐、Kafka 指标、推送指标。
2. [x] **压测可复现**：存在压测脚本（wrk 参数 + 负载均衡器配置 + 数据准备步骤），report 说明如何跑。
3. [x] **对比报告**：产出单机 vs 2 节点 vs 3 节点（经 LB）的吞吐（RPS）与延迟对比，记录环境与结论。
4. [x] **WS 单独测**：wrk 压 HTTP 接口（会话/好友/鉴权/消息），WS 长连接用轻量脚本单独测连接数上限，结果分开报告。
5. [x] **测试全绿**：metrics 埋点不破坏现有 `go test ./...`。

### 非目标

- **不做 OTel / 链路追踪**。
- 不做生产级 Grafana 面板（开发期轻量，指标 + 压测报告即可）。
- 不做管理后台。
- 不引入新的基础设施依赖（复用现有 Postgres / Kafka / compose）。

## 3. 约束与边界

- **metrics 库**：`prometheus/client_golang`（Go 生态标准，与 `slog` 共存）。
- **暴露方式**：独立端口（如 `:9090`）或 `/metrics` 路径——由设计决定，但**不得污染业务路由**。
- **可开关**：开发期应能通过环境变量（如 `METRICS_ENABLED`）按需开启，默认关闭或默认开启需在设计时定。
- **wrk 压测 HTTP**：wrk 只支持 HTTP 请求吞吐统计；WS 长连接压测需轻量脚本/工具单独处理。
- **LB 选择**：nginx 或 caddy（dev 单机即可），配置进版本库便于复现。

## 4. 交付物

| # | 交付物 | 归属 child |
|---|--------|-----------|
| 1 | Prometheus metrics 埋点 + `/metrics` 端点 | `08-01-obs-metrics` |
| 2 | wrk 压测脚本 + LB 配置 + 数据准备脚本 | `08-01-obs-loadtest-run` |
| 3 | 单机 / 2 节点 / 3 节点对比报告 | `08-01-obs-loadtest-run` |
| 4 | spec 更新：logging-guidelines.md 记录 metrics 约定 | `08-01-obs-metrics` |

## 5. 验收测试

- `cd backend && go test ./...` 全绿。
- `curl :9090/metrics`（或 `/metrics`）返回业务指标（HTTP 计数/延迟、WS 在线数、消息数）。
- 压测脚本 `make loadtest` 或 `scripts/loadtest.sh` 一键可跑。
- 对比报告存于 child 任务的 `research/` 或 `docs/`。

## 6. 非目标明细（防范围膨胀）

- 不加消息历史分片、不加跨节点在线数聚合 API。
- 不重构日志体系（slog 保留）。
- 不在前端加指标（前端观测是另一件事）。
