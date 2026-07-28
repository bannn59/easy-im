# State Management

> Where state lives in the easy-im frontend.

---

## Bootstrap status

Default: **React Query for server state + React Context for session/UI + realtime module for live events**. Introduce Zustand/Redux/Jotai only when cross-tree client state becomes painful and document the exception here.

---

## State categories

| Category | Examples | Home |
|----------|----------|------|
| Server state | Conversations, message pages, user profiles | TanStack Query cache |
| Realtime stream | Live message events, typing, presence | `realtime/` → merges into Query or tiny stores |
| Session | Auth token, current user, connection status | Auth/session context (memory; tokens per security policy) |
| View UI | Composer draft, emoji picker open, selected conv | Component state or feature store |
| URL state | Active `conversationId`, deep links | Router params / search params |

---

## Decision guide

```text
Is it from the server and cacheable?
  yes → React Query
Is it a live event?
  yes → realtime module, then project into Query/UI
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

1. HTTP history fetch hydrates Query cache (cursor pages).
2. WS `message.created` upserts by server `id` (and reconciles `client_msg_id`).
3. Receipts/edits/recalls apply patches to the same cache entries.
4. The list component reads ordered messages from cache/selectors — it does not own a second parallel array forever diverging from cache.

Avoid:

```text
Query cache ─── copy ───→ local useState messages ─── WS push only updates local
```

unless you have a single reducer that is clearly the owner (then Query may be history-only). Pick **one owner**.

---

## Session & security

- Access tokens: memory by default; prefer httpOnly cookies if backend supports that model.
- On 401: single refresh pipeline; eject duplicate refresh storms.
- Clearing session must close WS and clear Query cache.

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
- Keeping WS messages only in component state that unmounts on route change mid-chat.

---

## Common mistakes

1. Two sources of truth for the same message after optimistic send.
2. Forgetting to reset feature state when switching `conversationId`.
3. Global store becoming a dumping ground for server entities (use Query).

---

## Verification

- Switch conversations rapidly: no mixed messages from the previous chat.
- Offline → online: reconnect resync policy documented and tested.
