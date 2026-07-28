# Component Guidelines

> How React components are built in easy-im.

---

## Bootstrap status

Chat UI landed under `features/chat/`. Styling is **global CSS** in `src/styles/index.css` with minimal design tokens (`--ink`, `--surface`, `--line`, …). Do not introduce a second styling system without updating this file.

---

## Component layers

| Layer | Examples (current) | Rules |
|-------|--------------------|-------|
| Page / route | `AuthPage`, `HomePage` | Wire hooks, layouts; thin |
| Feature container | `ConversationRoom`, `AppShell` | Own data/effects; pass plain props down |
| Presentational | `MessageBubble`, `Composer`, `EmojiPicker`, `ReplyBar` | No fetching; props in, events out |
| App shell | `App` header + router | Providers and chrome only |

---

## Structure

Preferred order inside a component file:

1. Imports
2. Props type
3. Component function
4. Small helpers co-located only if not reused

```tsx
type MessageBubbleProps = {
  message: ChatItem;
  mine: boolean;
  senderLabel: string;
  showSender: boolean; // group only
  resolveSender: (id: string) => string;
  onReply: (m: ChatItem) => void;
  onRetry?: (m: ChatItem) => void;
};

export function MessageBubble(props: MessageBubbleProps) {
  // ...
}
```

- Prefer **named exports** for components (easier refactors than default exports).
- Props types are explicit; avoid `React.FC` unless the codebase standardizes on it.

---

## Composition

- Compose chat UI from small pieces (`MessageList` + `MessageBubble` + `Composer`).
- Use children/slots for layout chrome rather than boolean prop forests.
- Lift state only as far as needed (`ConversationRoom` owns messages/reply/draft).

---

## Chat bubble layout (executable)

**DOM structure** (required for horizontal avatar alignment):

```text
.bubble-item
  [.bubble-sender]          ← only when showSender (group + not mine)
  .bubble-row               ← avatar + bubble column; align-items: flex-start
    [.bubble-avatar]
    .bubble-col
      [.bubble-quote]
      .bubble
      .bubble-meta
    [.bubble-avatar--mine]
```

### Rules

| Rule | Detail |
|------|--------|
| Avatar ↔ bubble | Same flex row, **top-aligned**. Never put the sender name inside `.bubble-row` or use `margin-top` on the avatar to “catch up” with a name line. |
| Sender name | **DM** (`members.length ≤ 2`): hide on bubbles; room title shows peer name (or explicit conversation title). **Group** (`members.length > 2`): show name **above** `.bubble-row`, indented to the bubble column (`margin-left: avatar + gap`). |
| Theme | Layout may follow WeChat structure (list \| header / stream / composer). **Theme colors do not follow WeChat** — no brand green bubbles. Mine = dark `--ink` surface; theirs = light `--surface`. |
| Keys | `key={localKey ?? id}` for optimistic rows; never array index. |
| Status | `pending` / `sent` / `failed` on the row; failed exposes retry. |

### Wrong vs correct

```text
// Wrong — name inside row → avatar sinks relative to bubble
.bubble-row
  avatar
  col
    sender name
    bubble

// Correct — name outside row
.bubble-item
  sender name          (group only)
  .bubble-row
    avatar
    col → bubble
```

---

## Room layout

Workspace room is a **three-band column**: header (flex-shrink) → message list (`flex: 1; min-height: 0; overflow-y: auto`) → composer (flex-shrink). Do not use a fixed `max-height` on the list as the only height strategy; parent height chain must include `min-height: 0`.

Composer:

- `textarea`; **Enter** sends, **Shift+Enter** newline.
- Emoji inserts Unicode into `body` (no separate message type).
- Reply chip above input; send includes `reply_to_message_id` when set.

---

## Lists & performance (IM)

- Virtualize long message lists when history grows past a few hundred (not required for current 100-cap window).
- Stable keys as above.
- Memoize pure row components only when profiling shows cost.
- Keep heavy media lazy when media exists.

---

## Styling

**Chosen approach**: one global stylesheet `frontend/src/styles/index.css` + CSS variables on `:root`.

1. Extend tokens / BEM-ish class names in that file for chat chrome.
2. Do not mix Tailwind / CSS modules / styled-components in `features/` without a deliberate migration and this section rewrite.
3. Keep focus rings and contrast; do not strip a11y for aesthetics.
4. Never introduce WeChat green (`#07c160` etc.) as product chrome.

---

## Accessibility

- Interactive elements are buttons/links, not clickable `div`s without roles/keyboard support.
- Composer and dialogs manage focus sensibly; emoji picker closes on outside click.
- Live message regions: `aria-live="polite"` on the list is acceptable; avoid announcing every keystroke.
- Icon-only / emoji toolbar buttons need accessible names (`t('chat.emoji')`).

---

## Anti-patterns

- Fetching inside presentational bubbles.
- Prop drilling auth tokens — use session context/hook.
- Giant monolithic room file again under `app/`.
- Using index keys on timeline lists.
- Aligning avatars with `flex-end` on multi-line columns (breaks horizontal alignment).
- Showing peer id/email on every DM bubble when the header already names them.

---

## Common mistakes

1. Scrolling to bottom on every parent re-render, fighting user scroll-back — track near-bottom threshold (~80px).
2. Opening a new WebSocket per component mount — one `connectRealtime` per room effect.
3. Putting sender label inside the avatar row and papering over misalignment with `margin-top` on the avatar.
4. Forcing green “WeChat” skins when product tokens are minimal greyscale.
