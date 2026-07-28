# Thinking Guides

> **Purpose**: Expand thinking before coding so boundary and duplication bugs are caught early.

---

## Why thinking guides?

Most production IM bugs are not “hard algorithms”. They are missed boundaries:

- HTTP history and WS push disagree on message shape
- Client and server each invent an ordering rule
- The same ACL check is reimplemented three times and drifts
- Optimistic UI and server ACK diverge into duplicate bubbles

These guides force the right questions **before** implementation.

---

## Available guides

| Guide | Purpose | When to use |
|-------|---------|-------------|
| [Code Reuse Thinking Guide](./code-reuse-thinking-guide.md) | Find existing owners before adding copies | New helpers, DTO fields, repeated UI/logic |
| [Cross-Layer Thinking Guide](./cross-layer-thinking-guide.md) | Trace data across API, DB, MQ, WS, UI | Features spanning 2+ layers |

---

## Quick reference: thinking triggers

### Cross-layer

- [ ] Feature touches HTTP **and** WebSocket
- [ ] New message / receipt / presence field
- [ ] Ordering, cursors, or idempotency keys involved
- [ ] Cache (Redis/Query) plus durable store
- [ ] Multi-device or multi-gateway delivery

→ [Cross-Layer Thinking Guide](./cross-layer-thinking-guide.md)

### Code reuse

- [ ] About to copy a type, guard, or mapper
- [ ] Same constant in frontend and backend by hand
- [ ] Third call site reading the same untyped payload field
- [ ] New shared button/bubble that is 80% like an existing one

→ [Code Reuse Thinking Guide](./code-reuse-thinking-guide.md)

### Reviewing AI or hurried diffs

- [ ] Claimed security issue — is the data actually user-controlled?
- [ ] “Missing validation” — was it already enforced at the service ACL boundary?
- [ ] “Bug” in tests — would the test still pass if the feature were deleted? (tautology)

---

## Pre-modification rule (critical)

> **Before changing ANY wire field, error code, or topic name, search first.**

```bash
# examples once code exists
rg "client_msg_id" backend frontend packages
rg "message.created" backend frontend
rg "conversation.forbidden" backend frontend
```

Missed renames across Go DTOs, WS frames, and TS clients are the usual outage pattern.

---

## Bootstrap note

easy-im specs under `.trellis/spec/backend` and `.trellis/spec/frontend` are **bootstrap assumptions** until application code exists. When the first real modules land, update those specs and keep these thinking guides aligned with actual paths.

---

**Core principle**: thirty minutes of boundary thinking saves hours of duplicate-message and ghost-presence debugging.
