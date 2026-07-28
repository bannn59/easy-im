# Implement: Frontend i18n (en / zh-CN)

## Checklist

### 1. Dependencies

- [ ] 在 `frontend/` 安装 `i18next`、`react-i18next`（runtime deps，非 dev）。
- [ ] 确认 `package.json` / lockfile 更新。

### 2. i18n 基建

- [ ] 新增 `frontend/src/i18n/resolveLanguage.ts`（`easyim_lng` + navigator `zh*` 规则）。
- [ ] 新增 `frontend/src/i18n/locales/en.json` 与 `zh-CN.json`（同构 key；先列全量 key 再填中文）。
- [ ] 新增 `frontend/src/i18n/index.ts`：`init` + `setDocumentLang` + 导出 `i18n` / `SUPPORTED_LANGS` / `setAppLanguage`。
- [ ] 新增 `frontend/src/i18n/LanguageSwitcher.tsx`。
- [ ] `main.tsx` 顶部 `import './i18n'`。

### 3. 文案替换（按文件）

- [ ] `App.tsx`：导航、aria-label、Sign out；挂载 `LanguageSwitcher`。
- [ ] `HomePage.tsx`：eyebrow / title / lead / list / CTA / signed-in 文案。
- [ ] `AuthPage.tsx`：标题、切换提示、label、按钮、本地 `Request failed`。
- [ ] `HealthPage.tsx`：标题、说明、panel key、Checking…、启动 API 提示（命令本身可保持英文 code）。
- [ ] `AppShell.tsx`：侧栏、创建表单、列表空态、房间、composer、本地 fallback 错误、`You` / `Untitled` / loading 等。

### 4. 文档 lang

- [ ] init 与 `setAppLanguage` 时设置 `document.documentElement.lang`。

### 5. Validation

```bash
cd frontend && npm run typecheck
cd frontend && npm run build
```

手动（dev server）：

1. 无 `easyim_lng` + 浏览器中文 → 默认中文 UI。
2. 无 `easyim_lng` + 浏览器英文 → 默认英文 UI。
3. Header 切换 EN ↔ 中文，各页文案即时变化。
4. 刷新后语言保持。
5. 触发一次 API 错误：仍见服务端英文 message；断网/本地 fallback 为当前语言。
6. 消息正文、会话标题、email 不变。

### 6. Spec touch（实现后 / Finish 阶段）

- [ ] 可选：在 `.trellis/spec/frontend/index.md` 或 directory-structure 记一笔 `src/i18n/` 约定（Finish 的 update-spec 步骤处理，不阻塞本实现）。

## Risky files / notes

| 文件 | 风险 |
|------|------|
| `AppShell.tsx` | 文案点最多，改动面大；只换字符串，不改数据流 |
| `AuthPage` 句中 Link | 拆 key 或短句重组，避免硬插 HTML |
| JSON 同构 | 漏 key 时 i18next 回退到 key 或 en；实现后应用肉眼扫一遍中文页 |

## Rollback points

1. 依赖安装后、页面改造前：可卸依赖并删 `src/i18n/`。
2. 单页改造可按文件 revert。
3. 全量完成后若不满意：隐藏 Switcher + 强制 `lng: 'en'`。

## Out of implement scope

- 后端错误码词典
- 路由本地化
- 自动化 e2e（仓库尚无前端 e2e 基建；本任务以 typecheck/build + 手动为准）
