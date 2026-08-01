# Prometheus metrics 埋点与 /metrics

**任务**: `08-01-obs-metrics`  
**日期**: 2026-08-01  
**类型**: 复杂任务（child）— 埋点实现

## 1. 背景与问题

easy-im 后端仅有结构化日志（slog + `X-Request-ID`），无任何 metrics。压测与容量规划需要可量化的运行时指标。

## 2. 目标

集成 `prometheus/client_golang`，为 HTTP / WS / 消息 / Kafka / 推送路径埋点，暴露 Prometheus 文本格式的 `/metrics`。开发期轻量，可开关，不引入 OTel。

### 成功标准（验收）

1. **`GET /metrics`** 返回 Prometheus 文本格式（`promhttp.Handler`）。
2. **HTTP 指标**：请求计数 + 延迟直方图 + 状态码标签（middleware 埋点，覆盖所有路由）。
3. **WS 指标**：当前在线连接数（gauge，hub Register/Unregister 时增减）。
4. **消息指标**：发送/扇出计数（`message.created` 等事件的产生与投递）。
5. **Kafka 指标**：producer 发布成功/失败计数；consumer 消费计数（api fanout consumer + worker）。
6. **推送指标**（worker）：离线推送成功/失败计数。
7. **可开关**：环境变量（如 `METRICS_ENABLED`）控制；默认**开启但仅监听独立端口**（避免污染业务路由），或由设计定。
8. **不破坏现有测试**：`go test ./...` 全绿。

### 非目标

- 不做 OTel、不接 Grafana、不做业务自定义指标（只做路径级）。
- 不改日志体系。
- 不加前端指标。

## 3. 约束与边界

- **库**：`prometheus/client_golang`（`prometheus` + `promhttp`）。
- **埋点位置**：`internal/handler`（HTTP middleware）、`internal/hub`（WS 连接数）、`internal/mq`（Kafka producer/consumer 计数）、`internal/app`（fanout 消费）、`internal/push`（worker 推送）。
- **命名**：遵循 Prometheus 约定（`namespace_service_subsystem_*`），前缀如 `easyim_`。
- **标签**：`service`、`method`、`path`、`status`、`event_type` 等，与现有 slog 字段命名一致。
- **metrics 注册**：全局 `prometheus.DefaultRegisterer` 或独立 registry，由设计定。

## 4. 交付物

1. `internal/metrics`（新包）：注册指标、封装暴露。
2. `internal/handler`：HTTP metrics middleware。
3. `internal/hub`：在线连接 gauge 埋点。
4. `internal/mq`：producer/consumer 计数埋点。
5. `internal/app`：fanout 消费计数埋点。
6. `cmd/api` / `cmd/worker`：metrics 服务启动（独立端口）。
7. spec 更新：`logging-guidelines.md` 或新增 metrics 约定段。

## 5. 验收测试

- `cd backend && go test ./...` 全绿。
- `curl :9090/metrics`（或配置的端口）返回：HTTP 请求计数/延迟、WS 在线 gauge、消息计数、Kafka 计数。
- 压测时观察 `/metrics` 数值随负载增长（由压测任务验证）。

## 6. 非目标明细

- 不做指标保留/聚合策略（开发期）。
- 不加 metrics 认证（开发期，端口不暴露公网即可）。
