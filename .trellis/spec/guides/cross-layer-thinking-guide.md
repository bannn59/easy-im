# Cross-Layer Thinking Guide

> **Purpose**: Think through data flow across layers before implementing IM features.

---

## The problem

Most IM bugs happen at **boundaries**, not inside a single function:

- API returns camelCase chronologically; WS pushes snake_case with different ordering
- DB commits a message; MQ publish fails; other devices never see it
- Gateway trusts a frame and skips ACL that HTTP enforces
- Frontend optimistic row never reconciles with server id → duplicates

---

## Before implementing cross-layer features

### Step 1: Map the data flow

Draw the path end-to-end:

```text
Client composer
  → WS or HTTP
  → handler/ws decode
  → service (ACL, idempotency, seq)
  → repo (Postgres transaction)
  → commit
  → MQ publish
  → gateway fan-out / worker push
  → other clients realtime + cache merge
  → UI projection
```

For each arrow ask:

- What is the exact shape?
- Who validates?
- What happens on partial failure?
- Is the operation idempotent under retry?

### Step 2: Identify boundaries

| Boundary | Common issues |
|----------|---------------|
| Client ↔ HTTP | Auth headers, cursor pagination, error codes |
| Client ↔ WS | Frame version, heartbeat, reconnect resume |
| Handler/WS ↔ Service | DTO vs domain types, ACL |
| Service ↔ Repo | Transactions, `client_msg_id` uniqueness |
| Service ↔ MQ | Publish-after-commit, event versioning |
| MQ ↔ Gateway | At-least-once fan-out, de-dupe on device |
| Repo ↔ Redis | Presence not history; TTL/rebuild |
| Backend ↔ Frontend | Field names, time formats, id types |

### Step 3: Define contracts

For each boundary document:

- Input / output types
- Error codes
- Ordering and idempotency keys
- Compatibility (who tolerates unknown fields)

---

## IM-specific boundary rules

### 1. Single use-case owner

Sending a message — whether via HTTP or WS — must hit the **same** `service` method for ACL, quota, and persistence rules.

### 2. Ordering

Pick one server-side order key (`seq` per conversation recommended) and use it in:

- DB index
- HTTP history API
- WS payload
- Client sort / merge

### 3. New message fields (e.g. reply_to)

When adding a field to messages:

- [ ] Migration + domain + repo scan/insert
- [ ] Send validation (cross-conversation? deleted target?)
- [ ] HTTP DTO **and** WS `message.created` payload stay isomorphic
- [ ] List hydration batched (no N+1)
- [ ] Frontend `api/messages.ts` type + optimistic local row + bubble render
- [ ] Do **not** smuggle structured data only inside `body` text

→ Details: [Realtime & Messaging — message.reply_to](../backend/realtime-messaging.md), [Database — messages table](../backend/database-guidelines.md)

### 4. Optimistic UI

- Client `client_msg_id` is the reconcile key across pending HTTP and WS.
- One bubble per id/`client_msg_id`; retry reuses the same client id.
- Client list sort

Never let clients invent a global order from local clocks alone.

### 3. Idempotency

`client_msg_id` (per sender/device) is part of the **cross-layer contract**:

- Client retries with the same id
- DB unique constraint or equivalent
- WS/HTTP both accept and return the same server message
- UI reconciles optimistic row by that id

### 4. Publish after commit

```text
DB commit success → then MQ / local push
```

If publish must be reliable, use an **outbox** row in the same transaction, consumed by `worker`. Do not hold DB transactions open while writing sockets.

### 5. At-least-once delivery ⇒ client de-dupe

Gateways and workers may deliver twice. Clients **upsert by server message id**.

### 6. Presence is ephemeral

Redis presence must not become the only ACL or history store. After flush, users reappear on reconnect; history still loads from Postgres.

---

## Common cross-layer mistakes

### Mistake 1: Implicit format assumptions

**Bad**: One side uses Unix seconds, the other milliseconds; one side omits timezone.

**Good**: Explicit RFC3339 strings or documented epoch units in the contract.

### Mistake 2: Scattered validation

**Bad**: “Light” checks in WS, “real” checks in HTTP.

**Good**: Transport checks shape; service enforces business rules once.

### Mistake 3: Leaky abstractions

**Bad**: React component knows SQL column names or Redis key layouts.

**Good**: UI knows view models; `api/` / `realtime/` know wire types; `repo` knows SQL.

### Mistake 4: Every consumer parses the same payload

**Bad**: Each command casts JSON fields locally.

**Good**: One decoder per protocol surface (HTTP response parser, WS frame codec) exports typed values.

### Mistake 5: Dual write without recovery

**Bad**: Write Postgres and Redis independently; hope they match.

**Good**: Durable truth in Postgres; Redis rebuildable; document rebuild job.

---

## Checklist for cross-layer features

Before implementation:

- [ ] End-to-end flow drawn (including failure branches)
- [ ] Transport choice justified (HTTP vs WS vs both)
- [ ] ACL ownership named (`service` method)
- [ ] Idempotency key named
- [ ] Ordering key named
- [ ] Event / DTO version story named

After implementation:

- [ ] Round-trip test: create via one transport, read via the other
- [ ] Duplicate delivery does not duplicate UI rows
- [ ] Not-a-member denied on **both** transports
- [ ] Partial failure (MQ down) behavior is defined (outbox / error / metric)
- [ ] Frontend and backend error codes align
- [ ] No raw payload casts outside codec/guard modules

---

## Contract change checklist

When adding a field to a message, receipt, or presence event:

- [ ] Backend domain + repo + migration if persisted
- [ ] HTTP DTO / OpenAPI
- [ ] WS frame codec + version note
- [ ] MQ event payload
- [ ] Frontend TS type + guard
- [ ] UI projection / cache merge
- [ ] Tests on at least two layers
- [ ] Spec update if a new convention appeared

Search before rename:

```bash
rg "old_field_name" backend frontend packages
```

---

## Multi-node gateway checklist

- [ ] Local conn registry is node-local by design
- [ ] Cross-node delivery path exists (MQ/pubsub)
- [ ] Tests or a recorded manual plan for A on node1, B on node2
- [ ] Sticky routing is a hint, not the only correctness mechanism

---

## When to write a flow doc

Create a short flow note (in the task or `docs/`) when:

- Feature spans 3+ layers
- Delivery semantics are subtle (exactly-once illusions, offline push)
- Multiple devices or gateways are involved
- The feature already caused one production/debug incident

---

## Bootstrap note

Until application code exists, use the backend/frontend specs under `.trellis/spec/` as the intended architecture. After scaffolding, replace examples in those specs with real paths and keep this guide’s checklists.
