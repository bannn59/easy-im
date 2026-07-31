# Realtime & Messaging

> WebSocket gateway, presence, and async bus conventions for easy-im.

---

## Bootstrap status

**Landed (single process dev)**: `cmd/api` serves HTTP auth/conversations/messages and `GET /v1/ws`. In-process `internal/hub` fans out `message.created`, `message.read`, `typing.started`/`typing.stopped` to member user connections. Inbound WS frames (`typing.start`/`typing.stop`) are parsed and dispatched. No separate `cmd/gateway` / MQ yet — package boundaries should still allow a later split.

Message send path today: **HTTP only** for writes; WS is **bidirectional** (typing commands inbound, push events outbound). Clients de-dupe by `id` / `client_msg_id`.

---

## Process split

| Process | Role | Status |
|---------|------|--------|
| `cmd/api` | Auth, REST/JSON, history, **dev hub WS** | Landed |
| `cmd/gateway` | Maintain WS connections at scale | Planned |
| `cmd/worker` | MQ fan-out, offline push, jobs | Planned |

A single binary is acceptable for local dev **only** if the package boundaries still match the split above.

---

## WebSocket contract

### Lifecycle (current)

```text
HTTP Upgrade /v1/ws (cookie-auth) → hub register(userID) → push frames → close
```

- Authenticate before treating the conn as a user socket: read the `easyim_session`
  HttpOnly cookie (same JWT as HTTP auth). **No `?token=` query param** — it leaks into
  logs/history.
- `CheckOrigin` must validate the WS `Origin` against the CORS allowlist (cross-site
  WS would otherwise ride the cookie).
- Hub maps `userID → set of *Client` (multi-device ready).
- Reconnect is client-driven with backoff (`frontend/src/realtime`).
- Inbound client frames are parsed by `Hub.ReadPump` and dispatched via `Hub.FrameHandler`.
  Currently supported inbound types: `typing.start`, `typing.stop`.
- Server auto-expires typing state after 3 seconds (`Hub.SetTyping` / `ClearTyping`).

### Frame envelope (current)

**Server → Client:**

```json
{ "type": "message.created", "payload": { /* message DTO */ } }
{ "type": "message.edited", "payload": { /* message DTO with edited_at */ } }
{ "type": "message.recalled", "payload": { /* message DTO with recalled_at */ } }
{ "type": "message.read", "payload": { "conversation_id", "reader_id", "last_read_seq" } }
{ "type": "typing.started", "payload": { "conversation_id", "user_id" } }
{ "type": "typing.stopped", "payload": { "conversation_id", "user_id" } }
{ "type": "presence.changed", "payload": { "user_id", "online" } }
```

**Client → Server:**

```json
{ "type": "typing.start", "payload": { "conversation_id" } }
{ "type": "typing.stop", "payload": { "conversation_id" } }
```

Future versioning may add `v` / `request_id`; consumers must ignore unknown fields.

### What belongs on WS vs HTTP

| Use WebSocket | Use HTTP |
|---------------|----------|
| Live message push | History pagination / search |
| Typing indicators (inbound + relay) | Login, token refresh, **send message** (current) |
| Read receipt broadcasts (triggered by HTTP `POST .../read`) | Large media upload (object storage + HTTP) |
| Live presence changes (friend-scoped) | **Presence initial state** (`GET /v1/friends` includes `online`) |
| Lightweight client commands that need low latency | Presence / settings queries (future) |

Do not force all CRUD through WS.

---

## Delivery model

1. Client sends message via **HTTP** `POST /v1/conversations/{id}/messages`.
2. `MessageService` validates membership, body, optional reply target, assigns id; repo allocates `seq` and inserts.
3. After insert, hub publishes `message.created` to all conversation member user IDs (including sender).
4. At-least-once under reconnect: clients de-dupe by server message `id` and `client_msg_id`.

### Client idempotency

- Client supplies `client_msg_id` (unique per sender; max 128 bytes).
- Server unique `(sender_id, client_msg_id)`; duplicate insert returns the **first** stored row (including its original `reply_to_message_id`).

---

## Scenario: message.reply_to (code-spec)

### 1. Scope / Trigger

Cross-layer contract: optional quote/reply on text messages. Touches DB column, send/list HTTP, WS payload, and frontend bubble/composer. Do not implement reply as a body text prefix.

### 2. Signatures

**DB**

```sql
messages.reply_to_message_id UUID NULL
  REFERENCES messages (id) ON DELETE SET NULL
```

**HTTP send**

`POST /v1/conversations/{conversation_id}/messages`

**HTTP list**

`GET /v1/conversations/{conversation_id}/messages?before_seq=&limit=`

**Domain**

```go
// internal/domain.Message
ReplyToMessageID *string // nil = no reply
```

**Service**

```go
type SendMessageInput struct {
  ConversationID, SenderID, Body, ClientMsgID string
  ReplyToMessageID string // optional empty
}
// Send / List return MessageView { Message, ReplyTo *ReplyPreview }
```

### 3. Contracts

**Send request**

| Field | Type | Constraints |
|-------|------|-------------|
| `body` | string | required, trim, 1…4000 runes |
| `client_msg_id` | string | required, trim, len ≤ 128 |
| `reply_to_message_id` | string | optional; empty/omit = none |

**Message DTO / WS `message.*` payload** (HTTP and WS **must match**)

| Field | Type | Notes |
|-------|------|-------|
| `id`, `conversation_id`, `sender_id` | string | UUIDs |
| `body` | string | full text; on recall the client renders a placeholder instead |
| `client_msg_id` | string | |
| `seq` | number | int64 |
| `created_at` | string | RFC3339 UTC |
| `edited_at` | string \| null | set when the message was edited |
| `recalled_at` | string \| null | set when the message was recalled |
| `reply_to` | object \| null | explicit null when none / target gone |

Edit/recall are own-message-only and limited to a 5-minute window after send (backend-enforced).

**`reply_to` object**

| Field | Type | Notes |
|-------|------|-------|
| `id` | string | target message id |
| `sender_id` | string | target sender |
| `body` | string | **display truncate ≤ 120 runes**; storage remains full body via FK |

### 4. Validation & Error Matrix

| Condition | Result |
|-----------|--------|
| Not a member | `not_found` (conversation) |
| Empty / too long body | `invalid_argument` |
| Missing / too long `client_msg_id` | `invalid_argument` |
| `reply_to_message_id` set but id missing | `invalid_argument` ("reply target not found") |
| Target exists in **another** conversation | `invalid_argument` ("reply target not in conversation") |
| Target deleted (FK SET NULL on old rows) | list/send views show `reply_to: null` |
| Duplicate `(sender_id, client_msg_id)` | 201/OK with **original** message (idempotent) |

### 5. Good / Base / Bad Cases

- **Good**: reply to a message in the same conversation → stored FK + preview on send/list/WS.
- **Base**: omit `reply_to_message_id` → `reply_to: null`, pure text path unchanged.
- **Bad**: reply id from another conversation or random UUID → rejected, no insert.

### 6. Tests Required

- Unit (`message_service_test`): valid reply; missing target; cross-conversation; idempotent resend; preview truncation at 120 runes.
- Assertion points: `Message.ReplyToMessageID`, `ReplyTo.Body` rune count, list hydration includes preview without N+1 in repo design.
- Manual / future integration: HTTP send with reply → second client WS payload includes same `reply_to`.

### 7. Wrong vs Correct

#### Wrong

```text
// Encode quote only in body
body = "> Alice: hi\n\nmy reply"
// List N+1
for _, m := range list { repo.FindByID(m.ReplyTo…) }
// WS payload omits reply_to while HTTP includes it
```

#### Correct

```text
// Persist reply_to_message_id; DTO embeds truncated reply_to
// List: FindByIDs(distinct ids) once, map onto views
// messagePayload(view) shared shape for hub + HTTP handler DTO
```

---

## Presence

- Presence is **ephemeral**. DB may store last_seen asynchronously (not yet implemented).
- Online = has at least one healthy gateway/hub conn.
- **Landed (single process)**: hub tracks the online set and fires `PresenceBroadcaster` on 0↔1 connection transitions. `IsOnline` / `OnlineUserIDs` expose the set. `presence.changed` is broadcast to the user's friends via `ListFriendIDs`.
- **Never use presence as an ACL or message-history source of truth.**
- Multi-node fan-out would go through MQ or Redis pub/sub (future).

---

## Message bus

### Topic / subject naming

```text
im.message.created
im.message.recalled
im.conversation.updated
im.presence.changed
im.push.offline
```

Keep event payloads versioned. Producers own schema; consumers tolerate unknown fields.

### Technology choice

| Stage | Prefer |
|-------|--------|
| Early / single region | **NATS** JetStream (or core NATS + careful durability story) |
| Large partition / long retention / multi-team consumers | **Kafka** |
| Simple work queues only | RabbitMQ acceptable for jobs, weaker as the canonical event log |
| **Current dev** | In-process hub only — no durable bus yet |

Document the choice in config and do not abstract “every bus ever” on day one. One adapter interface in `internal/mq` is enough when introduced.

### Producer rules

- Publish **after** DB commit (or via outbox table polled by worker).
- Include aggregate IDs and full message DTO fields needed by clients (including `reply_to`).
- Never publish partial domain state that consumers must reverse-engineer from multiple races without ordering keys.

### Consumer rules

- Idempotent handlers (natural keys: message `id` / `client_msg_id`).
- Bound retry + DLQ/poison handling for worker jobs.
- Gateway consumers should only do fan-out, not re-implement domain writes.

---

## Multi-node gateway

Assume horizontal gateway scale from the start of design (even if one node in dev):

1. Local conn table is **node-local**.
2. Cross-node delivery goes through MQ/pubsub.
3. Sticky info (user → node) can live in Redis but treat it as a hint; still publish globally for correctness under races.

---

## Security & abuse

- Authenticate upgrade; refresh identity on resume if tokens expire.
- Rate-limit sends and presence updates per user/device.
- Validate frame size limits; reject oversized payloads early.
- Do not log message bodies at info level by default (see logging guidelines).
- Membership/ACL checks on every send; reply target must be same conversation.

---

## Anti-patterns

- Writing to DB inside the WS read loop without timeouts/backpressure.
- Pushing to sockets from random packages — only hub/gateway owns conn write.
- Using Redis pub/sub alone as the durable cross-region bus for message history fan-out without a durable store.
- Silent frame schema changes (HTTP has `reply_to`, WS does not).
- Blocking the gateway event loop on slow SQL.
- Storing quotes only as body prefixes.

---

## Verification

```bash
cd backend && go test ./internal/service/ -count=1
cd backend && go test ./... -count=1
```

- Unit: idempotent send with duplicate `client_msg_id`; reply validation matrix above.
- Manual: dual-tab WS — user A sends with reply, user B receives payload with `reply_to`.
- Future: dual-gateway delivery when process split lands.
