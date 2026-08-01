# Design — Prometheus metrics 埋点与 /metrics

**任务**: `08-01-obs-metrics`  
**日期**: 2026-08-01

---

## 1. 目标形态

```
promhttp.Handler ──► :9090/metrics   (独立端口, api 与 worker 各起一个)
        ▲
        │ promhttp.Handler().ServeHTTP
   ┌────┴─────┐
   │ internal │  注册自定义 Collector
   │ /metrics │
   └──────────┘
```

- **api 进程**: `:9090/metrics` — HTTP 中间件、WS 连接、消息、Kafka producer/fanout 指标。
- **worker 进程**: `:9091/metrics` — Kafka consumer、推送指标。
- 独立端口（不污染业务路由），由环境变量 `METRICS_ADDR` 控制（默认 `:9090` / `:9091`）。

## 2. 依赖与版本

- `github.com/prometheus/client_golang` **v1.24.x**（当前可用 v1.24.1）。
- 使用标准库 `net/http` 起第二个 listener（复用现有 `slog`，不引第三方 logger）。

## 3. 指标清单

### 3.1 HTTP 中间件（`internal/handler`）

| 指标名 | 类型 | 标签 |
|--------|------|------|
| `easyim_http_requests_total` | Counter | `service, method, path, status` |
| `easyim_http_request_duration_seconds` | Histogram | `method, path` |

- **path 归一化**: 路由变量用 `{id}` 占位符（如 `/v1/conversations/{id}/messages`），避免高基数。
- **实现**: 新 `metricsMiddleware` 包在现有 middleware 链中（`router.go`），实际顺序 `RequestID → Metrics → Recover → CORS`。Metrics 在 Recover 之内：panic 被 Recover 转成 500 后 Metrics 仍能记录正确状态码（Recover 也包在 Metrics 之内，panic 不会被 Metrics 捕获）。
- `status` 取自 `ResponseWriter` 包装器（record `WriteHeader` code）。

### 3.2 WS / hub（`internal/hub`）

| 指标名 | 类型 | 标签 |
|--------|------|------|
| `easyim_ws_online_conns` | Gauge | 无（或 `service`） |
| `easyim_ws_connections_total` | Counter | `service` |
| `easyim_ws_online_users` | Gauge | 无 |

- `Register` / `Unregister` 时增减 gauge。
- `online_users` = `len(h.clients)`；`online_conns` = 所有 user 的 conn 数之和。
- Hub 保持零依赖：通过可选的 metrics 钩子（注入式，nil-safe）或直接引用（hub 已在 handler 依赖里）。

### 3.3 消息（`internal/service` / `internal/app`）

| 指标名 | 类型 | 标签 |
|--------|------|------|
| `easyim_messages_created_total` | Counter | `conversation_id`（可选，或去掉降基数） |
| `easyim_messages_sent_total` | Counter | 无 |
| `easyim_fanout_events_total` | Counter | `event_type` |
| `easyim_fanout_skipped_total` | Counter | `reason`（own_origin / not_member） |

- `message.created` 发送/落库后 +1。
- fanout consumer 处理每个事件 +1，跳过（own origin）时 `skipped` +1。

### 3.4 Kafka（`internal/mq`）

| 指标名 | 类型 | 标签 |
|--------|------|------|
| `easyim_kafka_publish_total` | Counter | `topic, result`(ok/err) |
| `easyim_kafka_publish_duration_seconds` | Histogram | `topic` |
| `easyim_kafka_consume_total` | Counter | `topic, group` |
| `easyim_kafka_consume_errors_total` | Counter | `topic, group` |

- **Producer**: 在 `Publish`（异步成功回调）与 `ProduceSync` 里埋点。
- **Consumer**: 在 consumer 的 handler 回调里埋点（`consumer.go` 的 `Handle` 函数）。

### 3.5 推送（`internal/push`，worker）

| 指标名 | 类型 | 标签 |
|--------|------|------|
| `easyim_push_sent_total` | Counter | `result`(ok/err) |
| `easyim_push_aggregated_total` | Counter | 无 |
| `easyim_push_aggregate_batches_total` | Counter | 无 |

- 在 `flusher.go` / `worker.go` 成功与失败路径埋点。

## 4. 模块划分

```
internal/metrics/
  registry.go     // 全局默认 registry + 便捷初始化
  http.go         // metrics HTTP server (独立端口, 启停)
  names.go        // 指标名/标签常量
```

埋点代码直接调 `prometheus.MustRegister` + `promauto` 声明指标，放在各包内（`internal/handler/metrics.go`、`internal/hub/metrics.go` 等），避免过度抽象。

## 5. 开关与配置

- `METRICS_ADDR` 环境变量（默认 `:9090`）— 设置则启动 metrics server，空则禁用。
- api 用 `:9090`，worker 用 `:9091`（不同进程同机不冲突）。
- **默认开启**（开发期要能随时看）；不设则不开，保持现有行为不变。

## 6. 埋点侵入性控制

- Hub 的 metrics 通过可注入的 `Metrics *prometheus...` 指针字段实现（nil-safe），**不引入 package-level 全局**，保持 hub 可测试性。
- handler 中间件独立包，`router.go` 一行接入。
- mq 的 producer 埋点通过可选 `OnMetrics` 回调（nil-safe），不侵入核心接口。

## 7. 兼容与回归

- 无 metrics 时（`METRICS_ADDR` 空）行为与现状**完全一致**（所有钩子 nil-safe）。
- `go test ./...` 全绿。
- 现有 handler 测试（httpx_test、cors_test 等）不应受中间件影响——中间件包在 `Recover` 之后、`RequestID` 之后，纯观测。
