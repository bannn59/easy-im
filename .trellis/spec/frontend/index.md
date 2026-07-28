# Frontend Development Guidelines

> Coding guidance for the React + TypeScript client of **easy-im**.

---

## Bootstrap status

> **These specs are bootstrap assumptions, not evidence from existing product code.**
>
> The repository currently has no `frontend/` application source. Rules below describe
> the intended first-pass SPA conventions for a monorepo IM product. Re-bootstrap or
> edit these files when real code lands so examples point at actual modules and tests.

| Assumption | Choice |
|------------|--------|
| UI library | React 18+ with TypeScript |
| Bundler / app shell | Vite (or equivalent) under `frontend/` |
| Server state | TanStack Query (React Query) for HTTP |
| Realtime | Dedicated WebSocket client module (not ad-hoc `new WebSocket` in components) |
| Client state | React state + Context for session/UI; avoid a global store until proven necessary |
| Styling | TBD by first UI PR — document the choice here when picked (CSS modules / Tailwind / etc.) |

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

**Language**: Spec documents are written in **English**. UI strings may be localized later.
