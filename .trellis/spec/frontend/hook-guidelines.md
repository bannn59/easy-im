# Hook Guidelines

> Custom hooks and data-loading patterns for easy-im frontend.

---

## Bootstrap status

Assumed libraries: **TanStack Query** for HTTP; a project-owned **realtime module** for WS. Hooks wrap those modules — components do not.

---

## Naming & placement

| Hook kind | Name | Lives in |
|-----------|------|----------|
| Feature data | `useConversationList`, `useMessages` | `features/<feature>/` |
| Session | `useSession`, `useAuth` | `features/auth` or `shared/hooks` |
| Realtime subscription | `useConversationRealtime` | `features/chat` or `realtime/` wrappers |
| UI-only | `useAutoScroll`, `useElementSize` | `shared/hooks` |

Always prefix with `use`. One primary concern per hook.

---

## HTTP data fetching

Pattern:

1. `api/` exports plain async functions (`listMessages(convId, cursor)`).
2. Query key factories live next to them.
3. Feature hooks call `useQuery` / `useMutation` and expose view-friendly data.

```ts
// bootstrap sketch
export function useMessages(conversationId: string) {
  return useQuery({
    queryKey: messageKeys.list(conversationId),
    queryFn: () => api.listMessages(conversationId),
    enabled: Boolean(conversationId),
  });
}
```

Rules:

- Do not call `fetch` inside components when an `api/` function exists or should exist.
- Mutations update cache explicitly (or via invalidation) — define the policy per resource.
- Errors surface as structured `{ code, message }` from API layer, not bare `Error` strings when the server sent a code.

---

## Realtime hooks

WS belongs behind `realtime/` client:

```text
useEffect → realtime.subscribe(conversationId, handler) → cleanup unsubscribe
```

Guidelines:

- Subscribe/unsubscribe in effects; never leak listeners across navigations.
- Handlers should be stable or depend on minimal deps; prefer routing events into Query cache or a small inbox store.
- On event `message.created`, **merge into** messages query cache by id (de-dupe), do not blindly append duplicates.
- Connection status (`connected` / `reconnecting` / `offline`) is exposed via a single session-level hook, not per bubble.

---

## Optimistic send

Bootstrap approach for composer:

1. Mutation/optimistic insert with temporary id + `client_msg_id`.
2. On ACK / push with server id, replace temp row.
3. On failure, mark row `failed` and allow retry **with the same** `client_msg_id`.

Keep optimistic logic in a hook (`useSendMessage`), not in the textarea component.

---

## Anti-patterns

- `useEffect` + `fetch` copy-pasted across pages.
- Multiple components each creating their own `WebSocket`.
- Hooks that return half of React Query’s API plus reinvented loading flags inconsistently.
- Silent empty catch in hooks.

---

## Common mistakes

1. Forgetting `enabled` guards when `conversationId` is empty during route transitions.
2. Invalidating the entire cache on every WS event.
3. Stale closures in long-lived WS handlers (always verify deps / refs).

---

## Verification

- Hook unit tests with mock Query client for merge/de-dupe behavior.
- Ensure subscribe cleanup is covered (no listener growth across route changes).
