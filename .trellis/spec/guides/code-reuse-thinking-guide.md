# Code Reuse Thinking Guide

> **Purpose**: Stop before creating new code — does an owner already exist?

---

## The problem

Duplication is the #1 source of inconsistency bugs in IM systems:

- Bug fixes land in the HTTP path but not the WS path
- Frontend and backend drift on field names (`clientMsgId` vs `client_msg_id`)
- Three components each cast the same event payload differently

---

## Before writing new code

### Step 1: Search first

```bash
rg "functionOrTypeName" backend frontend packages
rg "error code or topic string" backend frontend
```

### Step 2: Ask

| Question | If yes... |
|----------|-----------|
| Does a similar function exist? | Use or extend it |
| Is this pattern used on the other transport (HTTP vs WS)? | Share domain/service logic |
| Is this a wire field? | Own it in contracts / codec, not in each feature |
| Am I copy-pasting from another file? | **STOP** — extract |

---

## Common duplication patterns (easy-im)

### Pattern 1: Dual transport business logic

**Bad**: Validate membership in HTTP handler and again differently in WS loop.

**Good**: `service.SendMessage` is the single use-case; both transports call it.

### Pattern 2: Hand-copied DTOs

**Bad**: Go struct tags, OpenAPI, and TS `interface Message` edited separately.

**Good**: One contract source (`packages/contracts` or OpenAPI/proto) generates or documents both sides. If generation is not ready, still keep a **single markdown/schema owner** and update both clients in one PR.

### Pattern 3: Repeated payload field extraction

**Bad**:

```ts
const body = (ev as { payload?: { body?: string } }).payload?.body;
```

copied across chat, notifications, and search features.

**Good**: `isServerMessageEvent` + typed accessors in `frontend/src/realtime/`.

**Rule**: If the same untyped field is read in 2+ places, add a shared guard/normalizer before a third reader appears.

### Pattern 4: Parallel message lists

**Bad**: React Query holds history; component `useState` holds “live” messages; merge is ad hoc.

**Good**: One projection owner (usually Query cache) with upsert by id.

### Pattern 5: Error code strings

**Bad**: `"not a member"` English string compared in UI.

**Good**: Stable `conversation.forbidden` code from backend error map; UI switches on code.

---

## When to abstract

**Abstract when**:

- Same logic appears 3+ times
- Logic is easy to get subtly wrong (cursors, idempotency, ACL)
- Both API and gateway need it

**Do not abstract when**:

- Single use
- Trivial one-liner
- Abstraction would invent a framework heavier than the duplication

---

## Reducers and exhaustive handling

When state is derived from `type` / `action` / `kind` (WS frames, MQ events), prefer one `switch` with a `never` default over scattered `if`s.

```ts
switch (event.type) {
  case 'message.created':
    return applyCreated(state, event);
  case 'message.recalled':
    return applyRecalled(state, event);
  default: {
    const _exhaustive: never = event;
    return state;
  }
}
```

In Go, prefer typed handlers registered by event name over giant unstructured `map[string]any` switches at each consumer.

---

## Checklist before commit

- [ ] Searched for existing similar code on **both** backend and frontend
- [ ] No new hand-rolled DTO copy without updating the contract owner
- [ ] No repeated untyped payload extraction
- [ ] Constants / topic names / error codes defined in one place per layer
- [ ] HTTP and WS paths share service-layer rules where applicable

---

## Gotcha: asymmetric update paths

**Problem**: Init/scaffold creates files automatically, but a manual path (scripts, codegen, copy-paste docs) drifts.

**IM example**: Adding a WS frame field updates the Go codec test but not the TS guard; chat works on refresh (HTTP) and breaks on live push (WS).

**Prevention**: Any wire change PR lists **all consumers** (API DTO, WS codec, TS types, docs). Prefer a checklist section in the PR body.

---

## Gotcha: “temporary” helper that becomes the second source of truth

**Problem**: A small `formatMessage` in the feature folder later diverges from `shared/lib`.

**Prevention**: If a second feature needs it, move it immediately; do not wait for a third copy.
