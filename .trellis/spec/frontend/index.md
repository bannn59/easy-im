# Frontend Development Guidelines

> Coding guidance for the React + TypeScript client of **easy-im**.

---

## Bootstrap status

> Scaffold and early product UI live under `frontend/`. **Chat room** is implemented in
> `features/chat/` (bubbles, composer, emoji, reply). TanStack Query is still **not** the
> message owner — room-local state + realtime merge; prefer topic guides and code when they
> disagree with older bootstrap assumptions.

| Assumption | Choice |
|------------|--------|
| UI library | React 18+ with TypeScript |
| Bundler / app shell | Vite under `frontend/` |
| Server state | Room-local list today; TanStack Query intended later |
| Realtime | `src/realtime/` WebSocket client (`message.created`) |
| Client state | React state + Session context |
| Styling | Global CSS `src/styles/index.css` (minimal tokens; **no WeChat green**) |
| UI i18n | `i18next` + `react-i18next`; locales `en` / `zh-CN` |

---

## Guidelines index

| Guide | Description | Status |
|-------|-------------|--------|
| [Directory Structure](./directory-structure.md) | Feature folders; `features/chat` ownership | Source-backed |
| [Component Guidelines](./component-guidelines.md) | Bubble layout, room bands, styling | Source-backed |
| [Hook Guidelines](./hook-guidelines.md) | Custom hooks, data fetching, WS hooks | Bootstrap |
| [State Management](./state-management.md) | Optimistic merge, reply draft, scroll | Source-backed |
| [Type Safety](./type-safety.md) | Shared DTOs, guards, no `any` | Bootstrap |
| [Quality Guidelines](./quality-guidelines.md) | Forbidden patterns, tests, review | Bootstrap |

---

## How to use these guidelines

1. Read **Directory Structure** before adding features.
2. For chat list / message pane work, read **Component Guidelines** and **State Management**.
3. Keep protocol types aligned with backend message DTO (including `reply_to`) — see backend realtime-messaging scenario.

**Language**: Spec documents are written in **English**. Product UI strings use i18n (`en` / `zh-CN`).
