# Implement — Prometheus metrics 埋点与 /metrics

**任务**: `08-01-obs-metrics`  
**日期**: 2026-08-01

---

## 步骤（ordered checklist）

### Step 1: 依赖 + `internal/metrics` 包骨架

**改动**:
- `go get github.com/prometheus/client_golang@v1.24.1`。
- 新建 `internal/metrics/`：
  - `names.go`：指标名/标签常量（`easyim_` 前缀）。
  - `registry.go`：`NewServer(addr string, log *slog.Logger)` 起独立 HTTP server，`promhttp.Handler()`。
  - `http.go`：server 启停（goroutine + 优雅关闭）。

**验证**:
- `go build ./...`。
- 单测：`metrics` 包 server 启动后 `GET /metrics` 返回 200 与 `go_gc_*` 运行时指标。

### Step 2: HTTP 中间件

**改动**:
- `internal/handler/metrics.go`：`MetricsMiddleware`（Counter + Histogram，path 归一化 `{id}`）。
- `router.go`：在 `RequestID` 之后包上 `MetricsMiddleware`（nil-safe：无 metrics 时跳过）。

**验证**:
- 起 api 打一个请求，`curl :9090/metrics` 出现 `easyim_http_requests_total` 与 `easyim_http_request_duration_seconds`。
- 现有 handler 测试全绿（中间件 nil-safe）。

### Step 3: hub WS 连接指标

**改动**:
- `internal/hub/metrics.go`：`easyim_ws_online_conns` / `easyim_ws_online_users` gauge + `easyim_ws_connections_total` counter。
- `Hub` 增加 nil-safe `Metrics` 钩子字段；`Register`/`Unregister` 调用。
- 组装处（`app/api.go`）注入。

**验证**:
- 打开/关闭 WS 连接，`/metrics` 中 `easyim_ws_online_conns` 随动。
- `go test ./internal/hub/` 全绿。

### Step 4: 消息 + fanout 指标

**改动**:
- `internal/service/message_service.go`：发送成功后 `easyim_messages_sent_total` +1（nil-safe 注入）。
- `internal/app/fanout.go`：`FanoutHandler` 处理每事件 `easyim_fanout_events_total{event_type}` +1；跳过 own-origin 时 `easyim_fanout_skipped_total` +1。

**验证**:
- 发一条消息，`/metrics` 中 `easyim_messages_sent_total` 增长。
- 双节点下 fanout 计数正确（跨节点事件被消费、own 被跳过）。
- `go test ./internal/service/ ./internal/app/` 全绿。

### Step 5: Kafka producer/consumer 指标

**改动**:
- `internal/mq/producer.go`：`Publish` 成功回调 / 失败回调埋 `easyim_kafka_publish_total{topic,result}`。
- `internal/mq/consumer.go`：handler 埋 `easyim_kafka_consume_total{topic,group}` + error 计数。

**验证**:
- 有 Kafka 时发送消息，`/metrics` 中 `easyim_kafka_publish_total` 增长；worker 消费后 `easyim_kafka_consume_total` 增长。

### Step 6: 推送指标（worker）

**改动**:
- `internal/push/flusher.go` / `worker.go`：成功/失败路径埋 `easyim_push_sent_total{result}` + 聚合批计数。

**验证**:
- worker 发一次推送，`/metrics` 中 `easyim_push_sent_total` 正确。

### Step 7: 装配 + 开关

**改动**:
- `internal/config`：新增 `MetricsAddr`（env `METRICS_ADDR`）。
- `cmd/api/main.go`：默认 `:9090` 起 metrics server。
- `cmd/worker/main.go`：默认 `:9091` 起 metrics server。
- spec 更新：`logging-guidelines.md` 补 metrics 约定段。

**验证**:
- `METRICS_ADDR=` 时行为与现状一致（无 metrics server）。
- `go test ./...` 全绿。

### Step 8: 收尾

- `go vet ./...`。
- 压测任务中观察 `/metrics` 随负载增长（联调）。
