# Frontend Development Guidelines

> Coding guidance for the React + TypeScript client of **easy-im**.

---

## Bootstrap status

> Scaffold and early product UI live under `frontend/`. Several rows below still describe
> intended conventions (e.g. TanStack Query) that are not fully adopted yet — prefer the
> code and topic guides when they disagree with bootstrap assumptions.

| Assumption | Choice |
|------------|--------|
| UI library | React 18+ with TypeScript |
| Bundler / app shell | Vite under `frontend/` |
| Server state | TanStack Query (React Query) for HTTP (not fully wired yet) |
| Realtime | Dedicated WebSocket client module under `src/realtime/` |
| Client state | React state + Context for session/UI; avoid a global store until proven necessary |
| Styling | Global CSS in `src/styles/index.css` (minimal tokens; no CSS modules/Tailwind yet) |
| UI i18n | `i18next` + `react-i18next`; locales `en` / `zh-CN` under `src/i18n/` |

---

## Guidelines index

| Guide | Description | Status |
|-------|-------------|--------|
| [Directory Structure](./directory-structure.md) | Feature folders, shared UI, API/WS modules | Bootstrap |
| [Component Guidelines](./component-guidelines.md) | Components, props, composition, a11y | Bootstrap |
| [Hook Guidelines](./hook-guidelines.md) | Custom hooks, data fetching, WS hooks | Bootstrap |
| [State Management](./state-management.md) | Session, server cache, realtime inbox | Bootstrap |
| [Type Safety](./type-safety.md) | Shared DTOs, guards, no `any` | Bootstrap |
| [Quality Guidelines](./quality-guidelines.md) | Forbidden patterns, tests, review | Bootstrap |

---

## How to use these guidelines

1. Read **Directory Structure** before adding features.
2. For chat list / message pane work, also read **State Management** and **Hook Guidelines**.
3. Keep protocol types aligned with backend contracts (`packages/` OpenAPI/proto or shared schemas).

**Language**: Spec documents are written in **English**. Product UI strings use i18n (`en` / `zh-CN`); see [Directory Structure](./directory-structure.md) (`src/i18n/`) and [Quality Guidelines](./quality-guidelines.md).
