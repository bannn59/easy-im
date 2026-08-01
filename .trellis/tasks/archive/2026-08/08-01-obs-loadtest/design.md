# Design — 可观测性metrics与并发压测（父任务）

**任务**: `08-01-obs-loadtest`  
**日期**: 2026-08-01

---

## 1. 目标形态

```
                          ┌─ child: 08-01-obs-metrics ──┐
 易观测 (Prometheus /metrics)                              │
                          └──────────┬──────────────┘
                                     │ 埋点 (HTTP/WS/消息/Kafka/推送)
  并发压测 (wrk + nginx LB)    ┌──────▼──────┐
                              │  child:      │  单机 vs 2-3节点对比报告
                              │  loadtest-run│
                              └──────────────┘
```

**父任务角色**: 只做协调与交叉验收，不直接实现代码。两个 child 交付物可独立验证。

## 2. 任务依赖

- `08-01-obs-loadtest-run` **依赖** `08-01-obs-metrics` 的 `/metrics` 端点（压测时抓 metrics 佐证）。
- 因此执行顺序: **先 metrics 埋点，后压测**。在 child 的 implement.md 已写明该顺序。

## 3. 关键决策（父层拍板）

| 决策 | 结论 | 依据 |
|------|------|------|
| metrics 库 | `prometheus/client_golang` v1.24.x | Go 标准、与 slog 共存、当前可用 |
| 暴露方式 | 独立端口 `:9090`(api) / `:9091`(worker)，`METRICS_ADDR` 开关 | 不污染业务路由 |
| 压测工具 | `wrk` (HTTP) + 轻量脚本 (WS 连接数) | wrk 官方 wg/wrk；WS 需单独测 |
| 多节点 | 同机多端口(8081/8082/8083) + nginx LB(:8080) | 验证应用层扩展，避免跨机器部署成本 |
| WS 范围 | 只测连接数上限，不测帧吞吐 | wrk 能力边界 + IM 消息吞吐已由 HTTP 覆盖 |

## 4. 边界与兼容

- 无 metrics 时（`METRICS_ADDR` 空）行为与现状完全一致。
- 压测脚本不改业务代码，`go test ./...` 始终全绿。
- 复用现有 Postgres(:5433) / Kafka(:19092) / docker compose，不新增基础设施。

## 5. 非目标

- 无 OTel、无 Grafana、无管理后台、无前端指标。
