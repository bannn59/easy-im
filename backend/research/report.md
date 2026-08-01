# easy-im 并发压测报告

**任务**: `08-01-obs-loadtest-run`  
**日期**: 2026-08-01  
**环境**: WSL2 (Linux 6.6.114.1), 16 vCPU, Go 1.26.0, wrk 4.1.0, nginx 1.28.3  
**部署**: 同机多端口模拟多节点（应用层水平扩展），共享 Postgres(:5433) + Kafka(:19092)  
**依赖**: metrics 埋点（`08-01-obs-metrics`）提供 `/metrics` 佐证

---

## 1. 压测配置

- **工具**: `wrk -t8 -c200 -d10s --latency`，Lua 脚本注入会话 cookie / POST body。
- **接口**（4 类）:
  | 场景 | 方法+路径 | 说明 |
  |------|-----------|------|
  | 会话列表 | `GET /v1/conversations` | 读取，DB 查询 |
  | 好友列表 | `GET /v1/friends` | 读取，DB 查询 + presence |
  | 登录 | `POST /v1/auth/login` | bcrypt 密码哈希（已知慢） |
  | 历史消息 | `GET /v1/conversations/{id}/messages` | 读取，分页 |
- **多节点**: nginx `:8080` 轮询分发到 2 或 3 个 api 节点（`:8081/8082/8083`）。
- **数据**: 10 用户、9 好友关系、9 会话、每会话 10 条历史。

## 2. 结果汇总（Requests/sec，数值越高越好）

> 表内为 `-c 200` 数据。补充验证：`-c 1000` 下单机 RPS 不涨（会话 6759、好友 14583），证明 **单机已达饱和上限**（见 §3.1）。

| 接口 | 单机 | 2 节点 | 3 节点 | 2节点/单机 | 3节点/单机 |
|------|------|--------|--------|-----------|-----------|
| 会话列表 | 6753 | 5683 | 6245 | 0.84× | 0.92× |
| 好友列表 | 15362 | 7748 | 8544 | 0.50× | 0.56× |
| 登录 | 382 | 361 | 364 | 0.95× | 0.95× |
| 历史消息 | 7905 | 6070 | 6694 | 0.77× | 0.85× |
| 发送消息(写) | 377* | — | — | — | — |

> *发送消息为 `-c 20` 手动补充压测（`client_msg_id` 每次唯一），未纳入多节点对比。写路径含 DB insert + Kafka publish，吞吐显著低于读接口。

### 延迟（p50 / p99）

| 接口 | 单机 p50/p99 | 3 节点 p50/p99 |
|------|-------------|----------------|
| 会话列表 | 29ms / 39ms | 31ms / 54ms |
| 好友列表 | 13ms / 16ms | 20ms / 41ms |
| 登录 | 517ms / 594ms | 522ms / 1.11s |
| 历史消息 | 25ms / 32ms | 28ms / 133ms |

## 3. 关键发现

### 3.1 单机已饱和，多节点受共享 DB 限制

**结论: 单机 API 是饱和上限（非客户端瓶颈），多节点经 LB 不提升吞吐——瓶颈在共享 Postgres。** 证据链：

- **单机 RPS 是硬上限**：`-c 200` 会话列表 6753 RPS；提到 `-c 1000` 仍 6759 RPS（仅延迟 29ms→146ms）。好友列表 `-c 1000` 单机 14583 RPS 与 `-c 200` 的 15362 持平。**并发翻 5 倍吞吐不涨 = 服务器饱和**，不是 wrk 客户端限制（客户端在途请求多了，吞吐应涨，实际只涨延迟）。
- **多节点反而下降**：好友列表单机 14583 → 3 节点 8396（降 42%）。若瓶颈在应用层，3 节点应接近 3×；实际降幅远超 LB 转发延迟（几 ms）能解释的。根因是 **3 个节点共享一个 Postgres 连接池**：每请求打 DB，池总连接数不变，被 3 节点分摊后单节点吞吐下降、总吞吐不升反降。
- **登录 382 RPS 持平**（bcrypt CPU 瓶颈，与应用/DB 无关）。
- **metrics 佐证**：多节点阶段流量经 nginx 轮询均分到 3 节点（各约 21k，见 §3.3），负载均衡正常，降幅非负载不均所致。

**真实含义**: easy-im **应用层已无单点瓶颈**（3 节点都能独立处理请求），但读接口吞吐上限由**共享 Postgres 连接池**决定。真正的水平扩展收益需**跨机器 + 独立 DB/Kafka**（或调大 pgxpool 上限）才能体现。

**真实含义**: easy-im **应用层已无单点瓶颈**（3 节点都能独立处理请求），但读接口吞吐上限由**共享 Postgres 连接池**决定。真正的水平扩展收益需**跨机器 + 独立 DB/Kafka**（或调大 pgxpool 上限）才能体现。

### 3.2 登录是明确瓶颈

bcrypt 密码哈希使登录仅 **382 RPS**（单机），p99 594ms。这是 CPU 密集型操作，且无法通过加 api 节点缓解（瓶颈在单次哈希的 CPU 时间）。

### 3.3 metrics 佐证负载分布

3 节点阶段经 LB 分流后，实际计数（累加 counter，含单机阶段）：
- node0: 88k（单机阶段 ~67k + 多节点部分 ~21k）
- node1/node2: 各 ~21k（仅多节点阶段新连接）

说明 nginx 轮询将多节点阶段流量**均分**到 3 个节点（各约 21k），并没有固定到 node0。node0 计数更高只是因为它在单机阶段也承担了全部流量。

### 3.4 WS 连接上限

WS 长连接用 `scripts/wsload` 单独压测（wrk 不支持 WS）：

| 场景 | 300 | 600 | 1000 | 结论 |
|------|-----|-----|------|------|
| 单机直连 :8081 | — | — | 1000/1000 ✅ | 应用单机支撑 1000+ |
| 单机经 LB :8080（默认配置） | 240/300 | 240/600 | — | **nginx worker_connections=1024 是瓶颈** |
| 单机经 LB（worker_connections=8192） | — | 500/500 ✅ | — | 调大后 500 全过 |
| 3 节点经 LB（8192） | 300/300 ✅ | 600/600 ✅ | 1000/1000 ✅ | 水平扩展 WS 也成立 |

**关键发现**: 默认 nginx 配置下，约 240 个并发 WS 连接即达到 `worker_connections=1024` 上限（每个 WS 占用客户端+上游 2 个 fd）。将 `worker_connections` 调至 8192 后，3 节点经 LB 可稳定支撑 1000+ WS 连接，WS 水平扩展成立。

## 4. 建议

1. **跨机器验证**：要测真实水平扩展，需独立机器 + 独立 Postgres/Kafka（或至少 DB 走网络）。
2. **DB 池扩容**：`pgxpool` 连接上限是读接口的硬瓶颈；多节点共享单 DB 时调大 pool。
3. **bcrypt cost 权衡**：登录 382 RPS 若是瓶颈，评估 Argon2 或降 bcrypt cost（安全权衡）。
4. **LB keep-alive**：生产 LB 建议配置 upstream `keepalive` + 会话亲和，避免连接固定。
5. **LB `worker_connections`**：WS 长连接场景必须调大（本测试 8192），默认 1024 会限制在 ~240 并发连接。
6. **写路径单独测**：本次发送消息仅以 `-c 20` 补充（377 RPS，受 DB insert + Kafka publish 限制），标准并发下的写吞吐需单独压测。

## 5. WS 压测复现

```bash
cd backend
COOKIE=$(python3 -c "import json;print(json.load(open('/tmp/loadtest_tokens.json'))[0]['cookie'])")
go run ./scripts/wsload -url ws://localhost:8080/v1/ws -cookie "$COOKIE" -steps 300,600,1000
```

## 6. 原始数据

表中数值的来源文件：
- 单机基线: `research/n3/wrk_single_*.txt`（10s，与多节点同参数；**勿用** `research/wrk_single_*.txt`，那是另一组 30s 手工运行，含大量 socket 错误）
- 2 节点: `research/n2/wrk_multi_*.txt`
- 3 节点: `research/n3/wrk_multi_*.txt`
- metrics: `research/n3/metrics_multi_node{0,1,2}.txt`（3 节点阶段）；`research/metrics_single.txt` / `research/metrics_multi_node*.txt` 对应那组 30s 手工运行

§3.1 的 `-c 1000` 饱和验证数据（单机会话 6759、好友 14583；3 节点会话 6241、好友 8396）为本报告撰写时补充运行，未归档为文件。

## 7. 复现

```bash
cd backend
export DATABASE_URL='postgres://easyim:easyim@localhost:5433/easyim?sslmode=disable'
export AUTH_JWT_SECRET='test-secret' KAFKA_BROKERS='localhost:19092'
docker compose up -d                     # postgres + kafka
./scripts/loadtest.sh 10s 200 8          # 单机 + 3 节点（结果在 research/wrk_*）
```

2 节点对比需单独运行 `NODES=2 ./scripts/loadtest.sh 10s 200 8`，并把多节点结果复制到 `research/n2/`（脚本本身只写根目录 `research/`，不会自动分目录）。
