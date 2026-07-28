# State Management

> Where state lives in the easy-im frontend.

---

## Bootstrap status

**Current code** (chat room): local `useState` message array in `ConversationRoom` + session context + `realtime/connectRealtime`. TanStack Query is **not wired yet** for messages; when introduced, migrate with a single owner (see below) — do not run Query cache and a permanent parallel `useState` list.

Default long-term target remains: **React Query for server state + React Context for session/UI + realtime module for live events**. Introduce Zustand/Redux/Jotai only when cross-tree client state becomes painful and document the exception here.

---

## State categories

| Category | Examples | Home (current → target) |
|----------|----------|-------------------------|
| Server state | Conversations, message pages | Local fetch / `useState` → TanStack Query |
| Realtime stream | `message.created` | `realtime/` merges into room list (by `id` / `client_msg_id`) |
| Session | Auth token, current user | `Session` context |
| View UI | Composer draft, emoji open, reply target | Component state in `Composer` / `ConversationRoom` |
| URL state | Active `conversationId` | Router `/app/c/:id` |

---

## Decision guide

```text
Is it from the server and cacheable?
  yes → React Query (when adopted); until then one room-owned list
Is it a live event?
  yes → realtime module, then project into list/cache
Is it auth/session for the whole app?
  yes → Session provider
Is it one screen’s widget state?
  yes → useState / useReducer locally
Do 5+ distant components need the same client-only state with complex updates?
  yes → consider a small global store (document why)
```

---

## Messages as projected state

Message lists should be treated as **projections**:

1. HTTP history fetch hydrates the list (today: `setMessages` in room; later: Query cache pages).
2. WS `message.created` **merges** by server `id` and reconciles `client_msg_id` (see `mergeMessage` in `features/chat/types.ts`).
3. Optimistic send inserts `status: 'pending'` with `id: local:${client_msg_id}` and `localKey`; success/WS replaces the same row — never append a second bubble.
4. Failed send keeps the row with `status: 'failed'` and retry reuses the same `client_msg_id` for server idempotency.
5. Receipts/edits/recalls (future) patch the same entries.

Avoid:

```text
Query cache ─── copy ───→ local useState messages ─── WS push only updates local
```

Pick **one owner**.

### Reply draft

- `reply` state holds `{ id, sender_id, body, senderLabel? }` for the composer chip.
- Only server-backed message ids may be replied to (reject `local:…` / pending / failed).
- Clear reply on successful send, cancel, or conversation switch.

### Scroll ownership

- `stickToBottom` ref: force scroll on self-send and initial load; on WS only if distance-to-bottom < threshold (~80px).
- Do not scroll on every parent render.

---

## Session & security

- Access tokens: memory by default; prefer httpOnly cookies if backend supports that model.
- On 401: single refresh pipeline; eject duplicate refresh storms.
- Clearing session must close WS and clear message UI state.

---

## Presence & typing

- Ephemeral: OK in memory maps keyed by `userId` / `conversationId`.
- Always expire typing indicators client-side with timeouts.
- Do not persist presence to `localStorage` as truth.

---

## Anti-patterns

- Redux for every keystroke in the composer.
- Duplicating “unread count” in three stores without a single derivation path.
- Using URL query strings for secrets.
- Keeping WS messages only in component state that unmounts on route change mid-chat without cleanup.
- Optimistic row that ignores `client_msg_id` on HTTP/WS ack → duplicate bubbles.

---

## Common mistakes

1. Two sources of truth for the same message after optimistic send.
2. Forgetting to reset draft/reply/messages when switching `conversationId`.
3. Global store becoming a dumping ground for server entities (use Query when adopted).
4. Computing pending `seq` from a stale `messages` closure instead of functional `setMessages` updater.

---

## Verification

- Switch conversations rapidly: no mixed messages from the previous chat; reply chip cleared.
- Send → pending → server/WS merge: single bubble.
- Offline → online: reconnect resync policy (poll fallback currently 15s) still de-dupes.
