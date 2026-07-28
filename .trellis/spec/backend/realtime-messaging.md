# Realtime & Messaging

> WebSocket gateway, presence, and async bus conventions for easy-im.

---

## Bootstrap status

Assumptions for an IM monorepo. No gateway code exists yet; these rules define the first architecture cut.

---

## Process split

| Process | Role |
|---------|------|
| `cmd/api` | Auth, REST/JSON CRUD, history pull, admin |
| `cmd/gateway` | Maintain WS connections, heartbeats, local push |
| `cmd/worker` | Consume MQ: offline push, fan-out across gateway nodes, search index, webhooks |

A single binary is acceptable for local dev **only** if the package boundaries still match the split above.

---

## WebSocket contract

### Lifecycle

```text
HTTP Upgrade → auth (token / ticket) → bind user+device → heartbeat loop → recv/send frames → close
```

1. **Authenticate before** accepting application frames. Failed auth closes the socket with a defined code/reason.
2. **One connection registry** in `internal/gateway` maps `userID → deviceID → conn`.
3. Heartbeat / idle timeout must be explicit (server and client). Document intervals in one place.
4. Reconnect is client-driven; server must be idempotent for resume cursors / client msg IDs.

### Frame design

- Define a versioned frame envelope early, e.g. `{ "v":1, "type":"...", "request_id":"...", "payload":{...} }`.
- Separate **client request types** (send, ack, typing) from **server push types** (message, receipt, presence).
- Codec lives in `internal/ws`. Handlers decode once to typed structs; business code never parses raw maps.

### What belongs on WS vs HTTP

| Use WebSocket | Use HTTP |
|---------------|----------|
| Live message push | History pagination / search |
| Typing, presence, receipts stream | Login, token refresh |
| Lightweight client commands that need low latency | Large media upload (use object storage + HTTP) |

Do not force all CRUD through WS.

---

## Delivery model (bootstrap)

Recommended early model:

1. Client sends message via WS or HTTP.
2. `service` validates membership, assigns server `seq` / ID, writes Postgres.
3. After commit, publish `message.created` to MQ.
4. Local gateway node pushes to online local conns; other nodes receive via MQ/pubsub and push to their conns.
5. Offline devices: worker generates push notifications.

**At-least-once** push to devices is the default. Clients de-dupe by server message ID.

### Client idempotency

- Client supplies `client_msg_id` (unique per sender device).
- Server stores uniqueness on `(sender_id, client_msg_id)` to make retries safe.

---

## Presence

- Presence is **ephemeral** (Redis). DB may store last_seen asynchronously.
- Online = has at least one healthy gateway conn (or explicit sticky session rule).
- Broadcast presence changes through MQ or Redis pub/sub; do not loop all API pods on each heartbeat.

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

Document the choice in config and do not abstract “every bus ever” on day one. One adapter interface in `internal/mq` is enough.

### Producer rules

- Publish **after** DB commit (or via outbox table polled by worker).
- Include `event_id`, `occurred_at`, aggregate IDs, and payload version.
- Never publish partial domain state that consumers must reverse-engineer from multiple races without ordering keys.

### Consumer rules

- Idempotent handlers (store processed `event_id` or use natural keys).
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
- Membership/ACL checks on every send and on subscribe/join rooms if rooms exist.

---

## Anti-patterns

- Writing to DB inside the WS read loop without timeouts/backpressure.
- Pushing to sockets from random packages — only gateway owns `conn.Write`.
- Using Redis pub/sub alone as the durable cross-region bus for message history fan-out without a durable store.
- Silent frame schema changes without version bumps.
- Blocking the gateway event loop on slow SQL.

---

## Verification (when implemented)

- Unit: frame codec round-trip, idempotent send with duplicate `client_msg_id`.
- Integration: dual-gateway test — user A on node1, user B on node2, message delivers both ways.
- Load: heartbeat and idle disconnect behavior under reconnect storms.

```bash
cd backend && go test ./internal/ws/... ./internal/gateway/... ./internal/mq/...
```
