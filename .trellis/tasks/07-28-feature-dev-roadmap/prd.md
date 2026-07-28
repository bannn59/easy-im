# 功能开发路线图

## Goal

把 **功能地图校准结果** 与 **分阶段功能开发计划** 固化为任务内可导航文档，作为后续 Trellis 子任务拆分与排期的依据。  
**本任务不写产品代码。**

## Background

- 功能地图（归档）：`.trellis/tasks/archive/2026-07/07-28-feature-map-survey/research/` — 调研时 0 实现。
- 脚手架已完成：`backend/cmd/api` + `GET /healthz`，`frontend` 壳 + `/`、`/health`。
- 地图建议顺序：鉴权 → 会话 → 消息持久化 → gateway WS → MQ 扇出。
- Spec：`.trellis/spec/backend|frontend|guides`。

## Scope

### In scope

1. 校准功能地图分组的 **当前状态**（相对脚手架后）。
2. 写入分阶段路线图：P0–P6、里程碑 M0–M5、建议任务切分、风险/新人、非目标。
3. `research/index.md` + `research/roadmap.md`。
4. 「下一步默认开做」建议。

### Out of scope

- 实现业务 Phase 代码。
- 重写整份功能地图明细（链到归档即可）。
- 改变 spec 技术选型（仅可 notes 建议）。

## Constraints

- 区分已实现 / partial / 未做；spec ≠ 已上线。
- 按用户可感知垂直切片推进。
- 高风险（多节点 WS、MQ、presence、推送）不排在 HTTP 消息闭环之前必做。

## Acceptance Criteria

- [x] 存在 `research/index.md` 与 `research/roadmap.md`。
- [x] 含功能地图校准表（脚手架后）。
- [x] 含 P0–P6、出口标准、建议任务切分。
- [x] 含里程碑 M0–M5 与下一步默认开做。
- [x] 含高风险延后与适合新人项。
- [x] 无产品代码改动。

## Notes

- 文档任务已完成 research；可 Phase 3.4 提交后 finish-work。
