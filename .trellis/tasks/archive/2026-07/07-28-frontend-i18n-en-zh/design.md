# Design: Frontend i18n (en / zh-CN)

## Architecture

在前端引入轻量 i18n 层，**不改后端**、不改路由 path。

```text
main.tsx
  └─ import './i18n'          # side-effect init (before App)
  └─ <App />
       └─ SessionProvider / BrowserRouter
            └─ Header (LanguageSwitcher + nav t())
            └─ pages use useTranslation()
```

### Dependencies

| Package | Role |
|---------|------|
| `i18next` | 核心实例、资源、`changeLanguage` |
| `react-i18next` | `useTranslation`、组件重渲染 |

**不引入** `i18next-http-backend`（资源体积小，随 bundle 打包即可）。  
**不引入** `i18next-browser-languagedetector`：首次语言与持久化用约 15 行自研逻辑即可，避免多一依赖与 `zh` → `zh-CN` 映射歧义。

### Module layout

```text
frontend/src/i18n/
  index.ts              # create/init i18n, export instance + helpers
  resolveLanguage.ts    # localStorage + navigator → 'en' | 'zh-CN'
  locales/
    en.json
    zh-CN.json
  LanguageSwitcher.tsx  # Header 控件（也可放 shared/ui；与 i18n 同目录更内聚）
```

与现有 `api/`、`realtime/` 并列的横切基础设施目录，符合 “app shell 挂 provider、shared 放可复用” 的精神；页面继续留在 `app/`。

### Resource shape

单一默认 namespace `translation`（应用小，拆 ns 收益低）。

Key 按 UI 区域嵌套，例如：

```json
{
  "nav": { "home": "Home", "status": "Status", "workspace": "Workspace", "signIn": "Sign in", "register": "Register", "signOut": "Sign out", "primary": "Primary" },
  "home": { "eyebrow": "...", "title": "...", "lead": "...", "now": "Now", "next": "Next", "..." : "..." },
  "auth": { "...": "..." },
  "health": { "...": "..." },
  "workspace": { "...": "..." },
  "common": { "loading": "Loading…", "you": "You", "untitled": "Untitled", "requestFailed": "Request failed" }
}
```

- `en.json` 为 source of truth；`zh-CN.json` 键集合必须与 `en.json` **同构**。
- 插值：`t('home.signedInAs', { email })` 等。
- 链接夹在句子中的文案：拆成多段 key，或使用 `Trans`（仅在必要时；优先拆段以保持简单）。

### Language resolution

存储 key：`easyim_lng`（与现有 `easyim_access_token` 前缀一致）。

```text
resolveLanguage():
  1. localStorage[easyim_lng] ∈ {en, zh-CN} → 用之
  2. navigator.language / languages 任一以 "zh" 开头（大小写不敏感）→ zh-CN
  3. else → en
```

`i18n.init({ lng: resolveLanguage(), fallbackLng: 'en', resources, interpolation: { escapeValue: false } })`  
（React 已防 XSS；与 react-i18next 惯例一致。）

`changeLanguage(lng)` 包装：

1. `i18n.changeLanguage(lng)`
2. `localStorage.setItem('easyim_lng', lng)`
3. `document.documentElement.lang = lng`

初始化后立即设一次 `document.documentElement.lang`。

### UI integration

| 区域 | 改动 |
|------|------|
| `main.tsx` | `import './i18n'`（确保在 `App` 前初始化） |
| `App.tsx` Header | `useTranslation` + `<LanguageSwitcher />` |
| `HomePage` / `AuthPage` / `HealthPage` / `AppShell` | 硬编码串 → `t('...')` |
| `index.html` | 可保持 `lang="en"`；运行时由 JS 覆盖 |

**LanguageSwitcher**：两个互斥按钮或 segmented control 风格，文案固定为语言自称（`English` / `中文`），不随当前语言翻译语言名（避免选中态混淆）。需 `aria-label` / `aria-pressed` 以可访问。

### Error messages policy

| 来源 | 处理 |
|------|------|
| 本地 fallback（`Failed to load`、`Send failed`、`Request failed`…） | 进词典，用 `t()` |
| `ApiError.message`（服务端） | **原样** `setError(err.message)`，不包 `t()` |
| `Session` / dev-only throw（`useSession must be used within…`） | 可不翻译（开发断言） |

### Compatibility

- 无迁移：新 localStorage key，旧用户无 key → 走浏览器探测。
- 无 API 变更。
- 无路由变更。

### Trade-offs

| 选择 | 收益 | 代价 |
|------|------|------|
| 资源打进 bundle | 零额外请求、离线可用 | bundle 略增（两语本文案量很小） |
| 单 namespace | 实现简单 | 日后模块变大可能再拆 ns |
| 自研 resolveLanguage | 依赖少、zh 映射清晰 | 无 detector 插件生态（当前不需要） |
| 不本地化服务端错误 | 范围可控 | 中文 UI 下偶见英文 API 错 |

### Rollback

1. 去掉 `LanguageSwitcher` 与 `t()` 调用可回退，但成本高；更实际：保留 i18n，默认 `lng: 'en'` 并隐藏切换器。
2. 删除 `easyim_lng` 即可恢复“首次探测”行为。
