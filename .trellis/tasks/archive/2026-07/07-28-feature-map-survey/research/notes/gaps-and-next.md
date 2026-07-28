# 缺口与建议下一步

## 缺口（相对「可用 IM」）

| 能力 | 代码 | 备注 |
|------|------|------|
| 任何用户可打开的页面 | 无 | 无 frontend |
| 任何 HTTP/WS API | 无 | 无 backend |
| 数据模型 / 迁移 | 无 | 无 migrations |
| 自动化测试 | 无 | — |
| 部署描述 | 无 | 无 compose/k8s |
| 产品 README | 无 | 仅 `AGENTS.md`（Trellis） |

## 与 bootstrap spec 的差距

`.trellis/spec/` 描述的是 **目标约定**（Go monorepo、React、Postgres、Redis、MQ、WS）。  
功能地图 **不得** 将这些行当作已上线功能。本调研已全部降级为 `planned_only` 或 `not_found`。

## 建议落地顺序（非本任务范围）

1. 初始化 `backend/go.mod` + `cmd/api` 健康检查（最小可运行）。
2. 初始化 `frontend/` Vite React 壳 + 空路由表（便于下次地图有「入口」可挂）。
3. 鉴权最小闭环 → 会话 CRUD → 消息持久化 → gateway WS → MQ 扇出。
4. 每合并一个用户可感知切片，**更新本 research 或重跑功能地图**。

## 复扫触发条件

出现以下任一即应刷新地图：

- 新增 `backend/` 或 `frontend/` 源码
- 新增路由、OpenAPI、WS 帧类型
- 新增 migrations 或 e2e
