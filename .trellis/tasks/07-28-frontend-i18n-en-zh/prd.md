# Frontend i18n: English and Simplified Chinese

## Goal

让 easy-im 前端 UI 支持 **English** 与 **简体中文** 双语切换：用户可在界面选择语言，刷新后仍保持偏好；首次访问按浏览器语言推断。

## Background

- 前端：React 18 + TypeScript + Vite SPA（`frontend/`），路由 `react-router-dom`。
- 当前无 i18n 基建；`package.json` 无本地化依赖。
- 可见文案均为硬编码英文，集中在：
  - `frontend/src/app/App.tsx`
  - `frontend/src/app/HomePage.tsx`
  - `frontend/src/app/AuthPage.tsx`
  - `frontend/src/app/HealthPage.tsx`
  - `frontend/src/app/AppShell.tsx`
- 会话标题、消息正文、用户 email 等用户/服务端内容不翻译。
- 后端 `ApiError.message` 为英文服务端原文，前端原样展示。
- 前端 spec 已预留 “UI strings may be localized later”。

## Requirements

1. **语言**：`en`（English）、`zh-CN`（简体中文）。
2. **覆盖范围**：当前前端自有 UI 文案——导航、按钮、标签、标题、引导文案、空态、加载态、本地错误回退文案、placeholder、`aria-label` 等。
3. **语言切换**：在全局 Header 提供可访问的切换控件；切换后已挂载页面文案立即更新，无需硬刷新。
4. **偏好持久化**：用户选择写入 `localStorage`；之后访问优先使用已存偏好。
5. **首次默认（决策 A）**：无已存偏好时，若 `navigator.language`（或等价）以 `zh` 开头则用 `zh-CN`，否则 `en`；`fallbackLng` 为 `en`。
6. **文档语言**：切换语言时同步更新 `document.documentElement.lang`（`en` / `zh-CN`）。
7. **不改写**：用户生成内容与服务端 `ApiError.message` 原文不被 i18n 替换。

## Out of Scope

- 后端 API / 错误码多语言与 `Accept-Language`
- 将服务端错误 message 映射为本地化文案
- 用户生成内容（消息 body、会话 title、email）
- 除 `en` / `zh-CN` 外的其他语言
- 路由 path 本地化（`/login` 等保持英文）
- 复杂 ICU 日期/数字格式化（当前 UI 几乎不需要）
- 为尚未存在的功能预埋大量空 key

## Key Decisions

| 决策 | 选择 |
|------|------|
| 首次默认语言 | 跟随浏览器：`zh*` → `zh-CN`，否则 `en`；有 localStorage 则优先 |
| 切换入口 | Header 内简洁控件（如 EN / 中文） |
| 服务端错误 | MVP 不本地化，原样展示 |
| 技术栈 | 见 `design.md`（`i18next` + `react-i18next`，资源打包进 bundle） |

## Acceptance Criteria

- [ ] AC1：用户可在 UI 中在 English 与 简体中文之间切换。
- [ ] AC2：切换后导航、首页、登录/注册、健康检查、工作区（列表 / 房间 / composer）的界面文案均随语言变化。
- [ ] AC3：刷新页面后仍保持用户上次选择的语言。
- [ ] AC4：清除偏好后首次访问：浏览器语言为中文系时默认简体中文，否则 English。
- [ ] AC5：用户内容（消息、会话标题、email）与服务端原始错误消息不因 i18n 被错误改写。
- [ ] AC6：`document.documentElement.lang` 与当前语言一致。
- [ ] AC7：`frontend` 的 `npm run typecheck` 与 `npm run build` 通过。

## Risks / Notes

- 首页营销长文案需人工中译，注意语气与现有英文一致（克制、产品向）。
- 本地 fallback 错误串（如 `Failed to load`）纳入词典；`ApiError.message` 不纳入。
- 实现细节与文件落点见 `design.md` / `implement.md`。
