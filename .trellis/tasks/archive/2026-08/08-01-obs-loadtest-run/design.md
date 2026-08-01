# Design — wrk 并发压测与水平扩展基线

**任务**: `08-01-obs-loadtest-run`  
**日期**: 2026-08-01

---

## 1. 目标形态

```
wrk ──► nginx/caddy (:8080, LB) ──┬──► api node1 (:8081)
                                  └──► api node2 (:8082)
                                  └──► api node3 (:8083)  [3 节点时]

api nodeN 共享: Postgres (:5433) + Kafka (:19092)
每个 api nodeN 暴露 metrics :9090+N  (压测时抓取佐证)
```

- **单机基线**: wrk 直连 node1 `:8081`。
- **多节点**: wrk 打 LB `:8080`，LB 轮询分发到 2–3 个 api 节点。
- **多节点模拟**: 同一台机器跑 2–3 个 api 进程（`PORT=8081/8082/8083`），共享同一个 Postgres + Kafka——验证的是**应用层水平扩展**，不是跨机器。

## 2. 工具链

| 工具 | 用途 | 安装 |
|------|------|------|
| `wrk` | HTTP 压测 | `apt install wrk`（官方 `wg/wrk`） |
| `nginx` | 负载均衡器 | `apt install nginx`（dev 单机） |
| Go / Node 轻量脚本 | WS 连接数压测 | 项目内脚本 |

## 3. 压测接口与数据准备

### 3.1 接口清单（wrk）

| 场景 | 方法+路径 | 前置数据 | 备注 |
|------|-----------|----------|------|
| 会话列表 | `GET /v1/conversations` | 用户有会话 | 读取为主 |
| 好友列表 | `GET /v1/friends` | 用户有好友 | 读取为主 |
| 登录 | `POST /v1/auth/login` | 预置用户 | 含密码哈希（bcrypt 慢） |
| 发送消息 | `POST /v1/conversations/{id}/messages` | 用户+会话 | 写入，含 DB + Kafka |
| 历史消息 | `GET /v1/conversations/{id}/messages` | 会话有历史消息 | 读取为主 |

### 3.2 数据准备脚本

`scripts/prepare_data.go`（Go 程序，跑在 backend 下）：
- 创建 N 个用户（如 100），固定密码（如 `pass1234`）。
- 建立好友关系、创建会话、灌入 M 条历史消息（如每会话 50 条）。
- 输出测试用户 token（登录后拿 cookie），供 wrk 使用。

### 3.3 wrk 参数（单机/多节点一致）

```
wrk -t 8 -c 200 -d 30s --latency <URL>
```

- 固定 `-t 8 -c 200 -d 30s`，保证单机与多节点可比。
- 每个接口单独跑，避免相互干扰。
- 记录：Requests/sec、Latency avg/p50/p99、Socket errors、非 200 响应。

## 4. WS 连接压测（单独测）

wrk 无法压测 WS 长连接。用轻量脚本 `scripts/ws_load.go`：
- 打开 N 个 WS 连接（并发递增：50/100/200/500/1000）。
- 观察：连接成功率、握手延迟、维持稳定（心跳）下的断连率。
- **只测连接数上限**，不测帧吞吐（IM 消息吞吐已由 HTTP 发送接口覆盖）。

## 5. 多节点 LB 配置

`deploy/nginx-loadtest.conf`：
- upstream 指向 node1:8081 / node2:8082 / node3:8083。
- 轮询（round-robin）。
- 监听 :8080。
- 注：HTTP 会话（cookie）无粘性要求（登录 cookie 全局共享 DB），可无脑轮询。

## 6. 执行流程（`scripts/loadtest.sh`）

```
1. 准备数据    →  ./prepare_data.go (建用户/会话/消息, 输出 token 文件)
2. 起依赖      →  docker compose up -d (postgres + kafka)
3. 起 1 节点   →  PORT=8081  go run ./cmd/api &
4. 单机压测    →  wrk 各接口直连 :8081 → 记录
5. 起 2/3 节点 →  PORT=8082/8083 各起一个 + nginx LB :8080
6. 多节点压测  →  wrk 各接口打 :8080 → 记录
7. 汇总报告    →  生成 research/report.md 对比表
8. 抓取 metrics → 压测时 curl :909x/metrics 关键值
```

## 7. 报告结构（`research/report.md`）

| 段 | 内容 |
|----|------|
| 环境 | 机器规格、Go 版本、wrk 版本、部署方式 |
| 单机基线 | 各接口 RPS / p50 / p99 / 错误率 |
| 多节点对比 | 2 节点、3 节点经 LB 的同指标 |
| 扩展系数 | 多节点 RPS ÷ 单机 RPS（如 3 节点 → ~2.5x 说明接近线性） |
| 瓶颈分析 | 结合 metrics（CPU、WS 连接数、Kafka 发布）定位瓶颈 |
| 结论 | 水平扩展是否有效、下一步建议 |

## 8. 风险与注意

- **bcrypt 登录**是天然瓶颈：登录压测 RPS 会低，属预期，单独标注。
- **Kafka 单 broker**可能是多节点瓶颈（单机 worker 消费）。记录即可，不作为本任务修复目标。
- **同一台机器**跑多节点共享 CPU，多节点提升可能被机器资源上限掩盖——报告中如实标注。
- 数据准备要**每次全新重建**（删表重建或清库），避免会话/消息累积影响对比。
