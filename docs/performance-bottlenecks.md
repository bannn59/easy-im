# 性能瓶颈清单

> 记录 easy-im 并发压测（2026-08-01，`08-01-obs-loadtest`）发现的可优化瓶颈点。
> 完整数据见 `backend/research/report.md`。本清单用于后续优化排期与跟踪。

---

## 测量基线（概要）

环境：WSL2 16 vCPU / Go 1.26 / 单 Postgres / 单 Kafka。wrk 参数 `-t8 -c200 -d10s`。

| 接口 | 单机 RPS | 备注 |
|------|---------|------|
| 会话列表 | ~6.7k | 单机已饱和（`-c1000` 吞吐不涨） |
| 好友列表 | ~15k | 同上 |
| 登录 | ~380 | bcrypt 哈希 CPU 瓶颈 |
| 发送消息（写） | ~377 | DB insert + Kafka publish |
| WS 连接 | 1000+ | 修好 nginx 配置后 |

---

## 瓶颈清单

按影响优先级排序。每项含：现象 → 根因 → 建议 → 状态。

### 1. 🔴 共享 Postgres 连接池 — 水平扩展的最大障碍

- **现象**：3 节点经 LB 后读接口吞吐反降（好友列表 15.4k → 8.4k）。
- **根因**：所有 api 节点共享同一个 Postgres（连接池总量固定），多节点分摊同一池后单节点吞吐下降，总吞吐不升反降。
- **建议**：
  - 跨机器部署 + 独立 DB / 读写分离。
  - 或调大 `pgxpool` 连接上限（当前未显式配置，用 pgx 默认）。
  - 读接口加缓存（如 Redis 缓存会话列表/好友列表），减少 DB 压力。
- **状态**: ⬜ 待优化

### 2. 🔴 bcrypt 登录 — 单点 CPU 瓶颈

- **现象**：登录仅 ~380 RPS，p99 594ms，且加节点无法缓解。
- **根因**：`bcrypt` 密码哈希是 CPU 密集型（默认 cost），单次哈希占满 CPU。
- **建议**：
  - 评估换 Argon2id（可调内存参数，同样安全但可并行优化）。
  - 或降低 bcrypt cost（权衡安全性）。
  - 登录接口独立部署/限流（防止 CPU 打满影响其他接口）。
- **状态**: ⬜ 待优化

### 3. 🟡 单 Kafka broker — 扇出/推送的吞吐上限

- **现象**：多节点 fanout 与离线推送共享单 broker。
- **根因**：`im.messages` 单 topic 单 broker，生产/消费竞争。
- **建议**：
  - 压测发送消息时观察 Kafka 是否成为写路径瓶颈。
  - topic 分区扩展 + 多 broker 集群（KRaft 多节点）。
  - producer 批量/缓冲调优（当前 `MaxBufferedRecords=1000`）。
- **状态**: ⬜ 待优化（需先测写路径确认）

### 4. 🟡 nginx `worker_connections` — WS 长连接上限

- **现象**：默认配置下 WS 并发 ~240 即失败。
- **根因**：`worker_connections=1024` 默认值，每个 WS 占 2 fd（客户端+上游）。
- **建议**：已调至 8192（`deploy/nginx-loadtest.conf`）；生产按需调大 + `worker_rlimit_nofile`。
- **状态**: ✅ 已修复（本任务）

### 5. 🟡 共享同机 CPU — 压测环境限制

- **现象**：3 节点 + worker + nginx 同机争抢 16 核。
- **根因**：压测在单机模拟多节点，非生产隔离。
- **建议**：跨机器压测才能得到真实水平扩展数据。
- **状态**: ⬜ 待验证（跨机器）

### 6. 🟢 发送消息写路径 — 未充分压测

- **现象**：`-c20` 下 377 RPS（含 DB insert + Kafka publish）。
- **根因**：写路径未在标准并发下充分测，实际瓶颈（DB insert vs Kafka publish）未分离。
- **建议**：标准并发下单独压测写路径，对比单机/多节点。
- **状态**: ⬜ 待压测

---

## 优化排期建议

| 优先级 | 项 | 前置 | 预期收益 |
|--------|-----|------|---------|
| P0 | 跨机器压测（验证真实扩展） | 多台机器 | 确认瓶颈归属 |
| P1 | DB 池调大 + 读缓存 | 无 | 读接口吞吐提升 |
| P1 | 登录算法评估（Argon2/bcrypt cost） | 无 | 登录 RPS 提升 |
| P2 | Kafka 多分区/多 broker | 跨机器 | 扇出/推送吞吐 |
| P2 | 写路径标准压测 | 无 | 定位写瓶颈 |

---

## 复现

```bash
cd backend
export DATABASE_URL='postgres://easyim:easyim@localhost:5433/easyim?sslmode=disable'
export AUTH_JWT_SECRET='test-secret' KAFKA_BROKERS='localhost:19092'
docker compose up -d
./scripts/loadtest.sh 10s 200 8     # 单机 + 3 节点对比
```
