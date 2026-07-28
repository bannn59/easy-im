# Implement: 项目功能地图调研

## Checklist

1. [x] 扫描仓库：确认 `backend/`、`frontend/`、`packages/`、`apps/` 等是否存在；记录于 `research/method.md`。
2. [x] 收集入口候选：路由表、菜单、`cmd` 入口、HTTP 路由注册、WS 帧类型、OpenAPI/proto。
3. [x] 按用户可感知分组归类；无证据的分组不写「已实现」。
4. [x] 为每条功能填 schema 字段；路径用仓库相对路径。
5. [x] 写 `research/features/<group>.md`（或等价结构）。
6. [x] 写 `research/index.md`：导航、计数、高风险、新人友好。
7. [x] 写 `research/notes/gaps-and-next.md`（若为空地图，说明脚手架建议）。
8. [x] 自检 AC：无把 spec bootstrap 当已实现；无产品代码改动。

## Validation

```bash
# 产出存在
test -f .trellis/tasks/07-28-feature-map-survey/research/index.md
# 无产品源码被本任务修改（相对 main 应仅任务 research / 规划文件）
git status --porcelain
# 占位/臆造抽查
rg -n "TODO|TBD|假路由|as if implemented" .trellis/tasks/07-28-feature-map-survey/research || true
```

## Review gate before start

- PRD / design / implement 已齐 → 用户确认后 `task.py start`。
- 执行阶段只写 `research/`（及必要的任务内笔记），不改产品树。
