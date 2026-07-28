# Directory Structure

> How the React + TypeScript frontend is organized.

---

## Bootstrap status

Scaffold landed: Vite + React + TS under `frontend/` with `src/app`, `api`, `realtime` (placeholder), `features`, `shared`, `styles`.  
Product features (auth/chat/…) remain empty targets.

---

## Intended layout

```text
frontend/
├── package.json
├── vite.config.ts
├── index.html
├── src/
│   ├── main.tsx
│   ├── app/                    # shell: router, providers, layouts
│   ├── features/
│   │   ├── auth/
│   │   ├── conversation/       # chat list, create/join
│   │   ├── chat/               # message list, composer, receipts
│   │   ├── presence/
│   │   └── settings/
│   ├── shared/
│   │   ├── ui/                 # dumb presentational components
│   │   ├── lib/                # pure helpers (date, cursor, id)
│   │   ├── hooks/              # cross-feature hooks
│   │   └── types/              # only truly shared TS types (prefer contracts package)
│   ├── api/                    # HTTP client, query keys, REST functions
│   ├── realtime/               # WS client, frame types, reconnect, event bus
│   └── styles/
└── tests/                      # optional e2e / playwright
```

Feature folders own their UI, feature hooks, and local types. They import `api/` and `realtime/`, not the reverse.

---

## Ownership rules

| Area | Owns | Must not own |
|------|------|--------------|
| `features/*` | Screens and feature-specific components | Raw `fetch` / raw `WebSocket` |
| `api/` | Base URL, auth header injection, REST calls, Query key factories | React components |
| `realtime/` | Connection lifecycle, encode/decode, subscribe API | Message business rules duplicated from server |
| `shared/ui` | Reusable presentational widgets | Feature data fetching |
| `app/` | Router, auth gate, provider tree | Deep feature UI |

---

## Naming conventions

| Kind | Convention |
|------|------------|
| Components | `PascalCase.tsx` (`MessageList.tsx`) |
| Hooks | `useSomething.ts` |
| Feature folder | `kebab-case` or lowercase single word |
| Query keys | factory in `api/*Keys.ts` or co-located `keys.ts` |
| Tests | `*.test.ts(x)` co-located or under `__tests__/` |

---

## Cross-package contracts

Prefer generating or sharing DTO types from monorepo contracts:

```text
packages/contracts/   # OpenAPI, JSON Schema, or proto → generated TS
frontend/src/api/     # imports generated types
backend/              # source of truth for protocol
```

Do not hand-maintain divergent `interface Message` in three feature files.

---

## Adding a feature (checklist)

1. Create `features/<name>/` with entry route component.
2. Add API functions + query keys under `api/` if HTTP is needed.
3. Add frame handlers under `realtime/` if live events are needed.
4. Wire route in `app/` router.
5. Keep side effects in hooks, not in deep presentational components.

---

## Anti-patterns

- `components/` mega-folder with no feature boundaries.
- Importing from `features/a` into `features/b` freely (extract to `shared` or lift API).
- Placing WS reconnection logic inside a single chat component.
- Duplicating backend URL and token logic per feature.

---

## Verification

```bash
cd frontend && npm test          # or pnpm / yarn
cd frontend && npm run typecheck
```
