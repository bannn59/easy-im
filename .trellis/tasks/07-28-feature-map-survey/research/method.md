# 调研方法与覆盖范围

**Task**: `07-28-feature-map-survey`  
**Date**: 2026-07-28  
**Repo HEAD** (at survey): `main` @ `56d2d7f`（journal）/ 工作区含本任务文件

## 目标

从路由、菜单、页面、API、模块目录反推 **用户可感知** 的已有功能；无证据不写「已实现」。

## 扫描步骤

### 1. 顶层与候选应用目录

| 路径 | 结果 |
|------|------|
| `backend/` | **不存在** |
| `frontend/` | **不存在** |
| `apps/` | **不存在** |
| `packages/` | **不存在** |
| `services/` / `src/` / `server/` / `client/` / `web/` / `api/` / `cmd/` / `internal/` | **均不存在** |

### 2. 非工具链文件全集

排除 `.git`、`.trellis`、`.claude`、`.codex`、`.cursor`、`.agents` 后，仓库仅有：

- `AGENTS.md` — Trellis 说明块
- `.gitignore` / `.gitattributes`

**无** `package.json`、`go.mod`、`*.go`、`*.ts`、`*.tsx`、`*.vue`、OpenAPI、proto、migrations、compose。

### 3. 入口模式检索

在排除工具链目录后，对 `Route|Router|createBrowserRouter|HandleFunc|WebSocket|@Get|@Post` 等模式检索：**0 命中**（无可检索的产品源文件）。

### 4. 测试

无 `*_test.go`、`*.test.ts(x)`、e2e 目录。

### 5. 对照（非实现证据）

| 来源 | 用途 |
|------|------|
| `.trellis/spec/backend/*` | 计划中的 Go monorepo 分层（bootstrap 假设） |
| `.trellis/spec/frontend/*` | 计划中的 React 结构（bootstrap 假设） |
| `.trellis/spec/guides/*` | 思维指南，非产品功能 |
| 已归档 `00-bootstrap-guidelines` | 说明 spec 来源为栈假设，非代码反推 |

## 明确不扫作「IM 产品功能」的内容

- Trellis 工作流、skills、hooks、agents（见 `notes/non-product-surface.md` 附录）
- 平台模板与 CLI 脚本

## 结论（方法层）

**产品功能证据集为空。** 主地图应为空实现表；计划能力仅可标 `planned_only`，且必须指向 spec 而非源码。

## 复扫命令（代码落地后）

```bash
# 目录
ls backend frontend packages apps 2>/dev/null

# 前端路由
rg -n "createBrowserRouter|Routes|path:" frontend

# 后端 HTTP
rg -n "HandleFunc|chi\.|gin\.|echo\.|http\.Handle" backend

# WS 帧
rg -n "message\.created|client_msg_id|websocket" backend frontend

# 迁移与测试
ls backend/migrations 2>/dev/null
rg -l "_test\.go|\.test\.tsx?" backend frontend
```
