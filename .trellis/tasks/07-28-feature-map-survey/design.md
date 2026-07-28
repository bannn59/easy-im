# Design: 项目功能地图调研

## Approach

只读调研 + 结构化写作。单一执行者即可；可选 `trellis-research` 做目录扫描，主会话负责综合与 index。

## Evidence sources (priority order)

| 优先级 | 来源 | 期望发现 |
|--------|------|----------|
| 1 | `frontend/` 路由、菜单、页面组件 | 用户入口 |
| 2 | `backend/cmd/*`、`internal/handler`、`internal/ws`、OpenAPI/proto | API / 实时协议 |
| 3 | `backend/migrations`、repo 模型 | 表与持久化依赖 |
| 4 | 配置样例、compose、env example | 外部服务依赖 |
| 5 | `*_test.go`、`*.test.ts(x)`、e2e | 相关测试 |
| 6 | 根 README / 产品文档 | 命名与范围（次于代码） |
| — | `.trellis/spec/*` | **仅作「计划架构」对照**，不得当作已实现功能证据 |

## Output layout

```text
.trellis/tasks/07-28-feature-map-survey/research/
├── index.md                 # 导航 + 统计 + 风险/新人专节
├── method.md                # 扫描范围、命令、空目录清单
├── features/
│   ├── _template.md         # 单功能字段模板（可选）
│   └── <group-slug>.md      # 按用户可感知分组
└── notes/
    └── gaps-and-next.md     # 缺口与建议下一步（若需要）
```

## Feature record schema

```yaml
name: string
group: string                 # 用户可感知分组
status: implemented | partial | planned_only | not_found
entry:
  - kind: route | page | menu | command | ws | http | other
    value: string
code:
  - path: string
    symbol: string | null
deps:
  services: []                # redis, nats, ...
  tables: []
  config: []
tests: []
risk: low | medium | high | unknown
newbie_friendly: true | false | unknown
notes: string
evidence: string              # 一句话说明如何从代码推出
```

## Grouping heuristic (user-perceptible)

建议分组（有证据才建文件；无则在 index 标「无已实现项」）：

1. 账号与鉴权  
2. 会话 / 通讯录  
3. 消息收发与历史  
4. 回执 / 已读 / 输入中  
5. 在线状态 / 多端  
6. 通知 / 推送  
7. 设置 / 个人资料  
8. 管理 / 运维（若对用户或运营可见）  
9. 其它发现项  

## Risk / newbie labeling

| 标签 | 启发式（需代码支撑） |
|------|----------------------|
| 高风险/高耦合 | 跨 HTTP+WS+MQ+DB；全局 conn 表；无测试的鉴权/ACL；双写 |
| 适合新人 | 边界清晰、有测试、少副作用、纯展示或单一 CRUD |

无代码时全部 `unknown`，并在 index 解释。

## Non-goals in design

- 不生成假路由/假 API 列表「方便好看」。
- 不在本任务提交产品实现。
