# Quality Guidelines

> Frontend quality bar for easy-im.

---

## Bootstrap status

Apply once `frontend/` is scaffolded. Add concrete CI commands to this file when package scripts exist.

---

## Required patterns

1. Strict TypeScript.
2. Data access only through `api/` and `realtime/` modules.
3. Feature-scoped folders for chat surfaces.
4. Keys on message lists use stable server/client ids.
5. WS subscribe cleanup on unmount / conversation change.
6. User-visible errors mapped from stable error codes.

---

## Forbidden patterns

| Pattern | Why |
|---------|-----|
| Raw `WebSocket` in feature components | Duplicated reconnect/auth bugs |
| Token in `localStorage` without team security decision | XSS risk |
| `dangerouslySetInnerHTML` for message bodies | XSS unless sanitized HTML is an explicit product requirement |
| Index keys on timelines | Reorder/mismatch bugs |
| Silent `catch {}` | Lost failures |
| Committing `.env` secrets | Security |
| Disabling TypeScript checks to ship | Contract drift with Go backend |

---

## Testing requirements

| Layer | Expectation |
|-------|-------------|
| Pure lib | Unit tests |
| Hooks (send/merge) | Tests with mock Query + fake realtime |
| Components | Critical presentational tests (composer states, failed send) |
| E2E | Login → open conv → send → see message (when stack allows) |

Prioritize tests around **de-dupe**, **optimistic reconcile**, and **conversation switch isolation** — these break often in IM clients.

---

## Tooling (bootstrap)

```bash
cd frontend
npm run typecheck
npm run lint
npm test
```

Enable eslint rules for hooks deps and `no-explicit-any` early.

---

## Code review checklist

- [ ] No new raw fetch/WS outside `api/` / `realtime/`?
- [ ] Message cache merge de-dupes by id / `client_msg_id`?
- [ ] Conversation change resets or keys cache correctly?
- [ ] Error codes handled, not only toast(err.message)?
- [ ] a11y for new interactive controls?
- [ ] Types aligned with backend contract change?
- [ ] Specs updated if a new convention appeared?

---

## Performance checklist

- [ ] Long lists virtualized when needed
- [ ] Images lazy
- [ ] WS handler batches high-frequency events (typing)
- [ ] No accidental full-list remount on each presence tick

---

## Anti-patterns in reviews

- “Quick” copy of Message types into a fourth file.
- Giant context providers re-rendering the full chat tree every keystroke.
- Mixing design systems / styling approaches without updating this spec.
