# Implement — 可观测性metrics与并发压测（父任务）

**任务**: `08-01-obs-loadtest`  
**日期**: 2026-08-01

---

## 执行顺序（跨 child）

### Step 1: `08-01-obs-metrics` — 先做埋点

见 child `implement.md`。完成后验证:
- `go test ./...` 全绿。
- `curl :9090/metrics` 返回 HTTP/WS/消息/Kafka 指标。

### Step 2: `08-01-obs-loadtest-run` — 后做压测

见 child `implement.md`。完成后验证:
- `scripts/loadtest.sh` 可复现跑通。
- `research/report.md` 含单机/2节点/3节点对比与结论。

### Step 3: 交叉验收（父层）

- 压测报告中能引用 `/metrics` 实际抓取的数值（证明 metrics 在负载下工作）。
- metrics 埋点未破坏压测结果（中间件 nil-safe、无性能回归）。
- 两个 child 各自 `go test ./...` 全绿。

### Step 4: 收尾

- 两 child 分别 commit + archive。
- parent 记录 journal + archive。
- 可选: 将压测结论固化为下一阶段建议（另开任务）。
