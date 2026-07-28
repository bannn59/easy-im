# Component Guidelines

> How React components are built in easy-im.

---

## Bootstrap status

Assumptions for a chat-heavy SPA. Styling system is intentionally undecided until the first UI PR; update this file when chosen.

---

## Component layers

| Layer | Examples | Rules |
|-------|----------|-------|
| Page / route | `ChatPage`, `LoginPage` | Wire hooks, layouts; thin |
| Feature container | `ConversationSidebar`, `MessagePane` | Own data hooks; pass plain props down |
| Presentational | `MessageBubble`, `Avatar`, `IconButton` | No fetching; props in, events out |
| App shell | `AppLayout`, `AuthGate` | Providers and chrome only |

---

## Structure

Preferred order inside a component file:

1. Imports
2. Props type
3. Component function
4. Small helpers co-located only if not reused

```tsx
// bootstrap sketch
type MessageBubbleProps = {
  id: string;
  body: string;
  mine: boolean;
  status: 'sending' | 'sent' | 'failed';
  onRetry?: (id: string) => void;
};

export function MessageBubble({ id, body, mine, status, onRetry }: MessageBubbleProps) {
  // ...
}
```

- Prefer **named exports** for components (easier refactors than default exports).
- Props types are explicit; avoid `React.FC` unless the codebase standardizes on it.

---

## Composition

- Compose chat UI from small pieces (`MessageList` + `MessageBubble` + `Composer`).
- Use children/slots for layout chrome rather than boolean prop forests (`isSidebar && isCompact && ...`).
- Lift state only as far as needed; message list virtualization state stays near the list.

---

## Lists & performance (IM)

- Virtualize long message lists (e.g. `@tanstack/react-virtual` or equivalent).
- Stable `key={message.id}` — never use array index for messages.
- Memoize pure row components when profiling shows re-render cost; do not preemptively memo everything.
- Keep heavy media (image/video) lazy.

---

## Styling

Until a system is chosen:

1. Pick **one** approach in the scaffolding PR.
2. Document it in this section (replace this paragraph).
3. Do not mix three styling systems in `features/`.

Accessibility-related visibility (focus rings, contrast) must not be removed for aesthetics.

---

## Accessibility

- Interactive elements are buttons/links, not clickable `div`s without roles/keyboard support.
- Composer and dialogs manage focus sensibly.
- Live message regions: use polite live regions sparingly; prefer explicit UI for new-message notices to avoid screen-reader spam.
- Icon-only buttons need accessible names.

---

## Anti-patterns

- Fetching inside presentational bubbles.
- Prop drilling auth tokens through the tree — use session context/hook.
- Giant 1k-line `Chat.tsx` with list + composer + ws + search.
- Using index keys on timeline lists.
- Blocking the main thread with JSON-heavy work on every frame event (batch updates).

---

## Common mistakes

1. Scrolling to bottom on every parent re-render, fighting user scroll-back.
2. Opening a new WebSocket per component mount.
3. Rendering raw backend error strings without mapping to user-facing copy.
