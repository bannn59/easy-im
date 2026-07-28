# WeChat-like chat UI: layout, bubbles, emoji, reply

## Goal

把 easy-im 聊天室从脚手架文档流，升级为接近微信的成熟 **文本聊天体验**：固定三层布局、气泡消息流、成熟输入栏（多行 + 表情工具栏）、结构化引用回复。

用户价值：打开会话后能像即时通讯产品一样阅读与发送消息，而不是填表单。

## Background

- 前端聊天 UI 集中在 `frontend/src/app/AppShell.tsx`（`ConversationRoom`）：单行 input、扁平消息行、无乐观更新/时间/头像/工具栏。
- 样式：`frontend/src/styles/index.css` 极简 token；workspace 为侧栏 + 主区文档流。
- 后端消息契约（`messages` 表 / domain / DTO）：`id, conversation_id, sender_id, body, client_msg_id, seq, created_at`；无 reply、类型、附件。
- 发送：`POST /v1/conversations/:id/messages`，`{ body, client_msg_id }`；body 最长 4000 runes；`client_msg_id` 幂等。
- 列表：`before_seq` + `limit`（UI 目前一次 100，未分页加载更早历史）。
- 实时：WS `message.created`，payload 与 message DTO 同形；15s HTTP 轮询兜底。
- Spec 目标目录：`features/chat/`、`features/conversation/`（尚未落地）。
- 用户 scope：**B** = 布局 + 气泡 + 文本发送成熟化 + 表情工具栏 + 引用回复。
- 决策：**引用回复扩展后端**（可选 `reply_to_message_id` + 响应内嵌摘要），不把引用拼进 `body`。

## Requirements

### R1 — Workspace / room layout

- 桌面端：会话列表 | 聊天主区并排；主区为 **顶栏 / 可滚动消息区 / 底部输入栏**，主区占满可用视口高度（非随内容无限撑高的文档页）。
- 消息区独立滚动；发送成功、乐观插入、WS 收到本会话新消息时默认滚到底（用户明显上翻阅读历史时可不强制打断，见 design 细节）。
- 小屏可侧栏堆叠，但 room 内三层结构保持。

### R2 — Message bubbles

- 自己 / 对方左右气泡可区分。
- 展示基于 `created_at` 的可读时间；相邻消息可做轻量时间分隔（不必复刻微信完整算法）。
- 对方显示发送者短标识（邮箱 local-part 或现有 email）；自己侧不重复强调「你」。
- 头像用首字母/色块占位，不接真实头像系统。
- 引用块：有 `reply_to` 时在气泡内展示被引发送者 + 正文摘要。

### R3 — Mature text composer

- 多行 textarea；**Enter 发送**，**Shift+Enter 换行**。
- 空内容（trim 后）不发送；发送中防重复提交。
- 乐观更新：本地先插入 pending 气泡，成功后用服务端消息（按 `id` / `client_msg_id`）替换；失败标记可点重试。
- 错误对用户可见（沿用/扩展现有 err 展示）。

### R4 — Emoji toolbar

- 输入栏工具区提供表情入口；点选 Unicode emoji 插入光标处（或文末）。
- 固定 emoji 列表即可；不接 sticker 包、不上传自定义表情。
- Emoji 仍是文本 `body` 的一部分，不引入消息类型。

### R5 — Quote / reply（后端 + 前端）

- DB：`messages.reply_to_message_id UUID NULL`，FK → `messages(id)`，`ON DELETE SET NULL`。
- 发送可选 `reply_to_message_id`；服务端校验：目标存在且 `conversation_id` 相同，否则 `invalid`。
- HTTP 与 WS 的 message DTO 增加可选 `reply_to: { id, sender_id, body }`（目标已删则为 `null` 且不带悬空 id，或仅省略 — design 定一种）；`body` 可截断展示长度。
- UI：消息上可「回复」→ 输入区上方引用条（可取消）→ 发送后气泡展示引用。
- 旧客户端忽略未知字段仍可收发纯文本。

### R6 — Structure & i18n

- 聊天相关 UI 抽到 `frontend/src/features/chat/`（列表/气泡/composer/emoji）；`AppShell` 保留会话壳与路由装配，避免单文件堆叠。
- 新增文案走 `en` / `zh-CN`。
- 样式仍用全局 CSS + 现有 design tokens 扩展气泡表面色，不引入新 CSS 框架。
- **主题色不参考微信**：不用微信绿/品牌绿；布局与交互可对齐微信，配色延续本项目 minimal 黑白灰（可加中性气泡底），禁止为「像微信」而引入绿色主色。

## Out of scope

- 图片 / 文件 / 语音 / 视频与上传
- 已读回执、输入中、在线状态
- 撤回 / 编辑 / 转发 / 多选 / 长按菜单全集
- 会话列表未读角标、置顶、免打扰、last message 预览（API 尚无 last_message 时不做）
- 消息列表虚拟化与历史上翻无限加载（本轮保持当前窗口 + 滚到底；`before_seq` 可留接口不强制 UI）
- 独立 gateway 进程 / MQ 扇出改造
- 微信主题色 / 品牌绿 / 绿色气泡皮肤（布局可像微信，配色不跟）

## Constraints

- 保持现有 auth、会话 CRUD、文本收发、WS 推送可用。
- `client_msg_id` 幂等语义不变；带相同 `client_msg_id` 重试不得产生双条（含 reply 字段时以首次写入为准）。
- body 仍 ≤ 4000 runes；reply 摘要展示截断不影响存储全文。
- 迁移可逆（goose Down）；本地需 `migrate up` 后联调。
- 后端服务测试（`message_service_test`）与前端 `tsc`/既有检查需通过。

## Acceptance Criteria

- [ ] AC1：打开会话后主区为顶栏 + 可滚消息流 + 底栏输入，主区高度贴合工作区；新消息默认可见（滚到底）。
- [ ] AC2：自己与对方气泡左右可区分；可见时间；对方有发送者标识与头像占位。
- [ ] AC3：Enter 发送 / Shift+Enter 换行；空内容不发送；失败可见且可重试。
- [ ] AC4：发送有 pending → 成功替换（或等价乐观体验）；与 WS/HTTP 去重不双气泡。
- [ ] AC5：可打开表情面板，插入 emoji 后成功发送并在气泡中显示。
- [ ] AC6：可对消息点回复；输入区显示可取消的引用条；发送后气泡展示结构化引用（来自 API `reply_to`，非 body 前缀解析）。
- [ ] AC7：迁移后 Send/List/WS 均透传 `reply_to`；无效 `reply_to_message_id` 返回错误且不落库。
- [ ] AC8：主要聊天组件位于 `features/chat/`（或等价 feature 目录）；新增文案有 en/zh-CN。
- [ ] AC9：回归：注册/登录 → 建会话 → 双方收发纯文本与带引用消息仍可用。

## Key decisions

| 决策 | 选择 |
|------|------|
| Scope | B：布局 + 气泡 + 文本成熟化 + 表情 + 引用 |
| 引用存储 | 后端 `reply_to_message_id` + DTO 内嵌 `reply_to` 摘要 |
| 表情 | Unicode 插入 body，无独立消息类型 |
| 视觉 | 布局对齐微信结构；**主题色不参考微信、不用绿色**；延续 minimal 黑白灰 + 中性气泡 |
| 任务形态 | 单任务垂直切片（backend reply + frontend room），不拆父子 |

## Technical notes

- 后端触点：migration、`domain.Message`、`MessageRepo`、`MessageService`（含 store 接口与 mem fake）、handler DTO、WS broadcast payload、service tests。
- 前端触点：`api/messages.ts`、`realtime` 类型、`AppShell` 拆分、`styles/index.css`、i18n locales。
- 详细设计与步骤见同目录 `design.md`、`implement.md`。
