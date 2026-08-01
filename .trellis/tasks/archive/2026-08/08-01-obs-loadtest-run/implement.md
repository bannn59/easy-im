# Implement — wrk 并发压测与水平扩展基线

**任务**: `08-01-obs-loadtest-run`  
**日期**: 2026-08-01

---

## 前置依赖

本任务依赖 `08-01-obs-metrics` 的 `/metrics` 端点（压测佐证）。若未就绪，先完成该任务或先跳过 metrics 佐证段。

## 步骤（ordered checklist）

### Step 1: 工具安装与验证

**改动**:
- `apt install -y wrk nginx`。
- `wrk --version`、`nginx -v` 确认。

**验证**:
- wrk 可跑 `wrk http://localhost:8080/healthz`。

### Step 2: 数据准备脚本 `scripts/prepare_data.go`

**改动**:
- 建 N 用户（默认 100，固定密码）、互加好友、建会话、灌历史消息（每会话 50 条）。
- 输出 `scripts/test_tokens.json`（登录 cookie / token 供 wrk 脚本用）。

**验证**:
- 跑完后 DB 中有用户/会话/消息；登录能拿 cookie。

### Step 3: 单机基线压测

**改动**:
- `scripts/loadtest.sh` 单机段：`PORT=8081 go run ./cmd/api &`。
- wrk 各接口直连 `:8081`：会话列表、好友列表、登录、发消息、历史消息（`-t 8 -c 200 -d 30s --latency`）。
- 每接口记录 RPS/p50/p99/错误率。

**验证**:
- 每接口 wrk 完成，输出完整。
- 压测时 `curl :9090/metrics` 关键值可抓取。

### Step 4: 多节点 + LB

**改动**:
- `deploy/nginx-loadtest.conf`：upstream 轮询 node1:8081 / node2:8082 / node3:8083，监听 :8080。
- `loadtest.sh` 多节点段：起 2 节点（8081+8082）、3 节点（+8083），wrk 打 :8080。

**验证**:
- LB 轮询生效（不同请求落到不同节点日志确认）。
- 多节点跑同参数 wrk，输出完整。

### Step 5: WS 连接压测

**改动**:
- `scripts/ws_load.go`：并发递增（50/100/200/500/1000）开 WS 连接，记录成功率/握手延迟/断连率。
- 单机 + 3 节点（经 LB，注意 WS 经 LB 需配置 upgrade header 透传）。

**验证**:
- 脚本输出连接数上限与瓶颈。

### Step 6: 汇总报告 `research/report.md`

**改动**:
- 对比表：单机 / 2 节点 / 3 节点 × 各接口 RPS / p50 / p99 / 错误率。
- 扩展系数、瓶颈分析（结合 metrics）、结论。

**验证**:
- 报告完整，结论明确（扩展是否接近线性、瓶颈在哪）。

### Step 7: 收尾

- 清理压测进程（kill api/LB）。
- 确认测试数据不影响后续开发（用独立 DB 或清库说明）。
- `go test ./...` 不受影响（压测脚本不改业务代码）。
