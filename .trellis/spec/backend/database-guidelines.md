# Database Guidelines

> Persistence conventions for easy-im backend.

---

## Bootstrap status

Landed: Postgres via **pgx** pool (`internal/db`), **goose** SQL under `backend/migrations/`,
`cmd/migrate`, and local `docker-compose.yml` (host port **5433**).

Schema so far: `users`, `conversations`, `conversation_members`, `messages` (with `next_seq` on conversations), optional `messages.reply_to_message_id`, `friend_requests`, and `friendships`.

Repo pattern: `internal/repo` with explicit SQL; domain types in `internal/domain`.

---

## Stack choices

| Concern | Bootstrap choice |
|---------|------------------|
| Primary DB | PostgreSQL 16+ |
| Access | `database/sql` + carefully owned queries, **or** `sqlc` for typed SQL |
| Driver | `pgx` (preferred) via `database/sql` compatible mode or pgx pool |
| Migrations | Versioned SQL in `backend/migrations/` (`golang-migrate` or `goose`) |
| Cache | Redis — **not** a source of truth for message history |
| Hot counters / presence | Redis with explicit TTL and rebuild paths |

Avoid introducing a heavy ORM as the default. IM queries are often explicit (range scans by conversation, cursor pagination, fan-out indexes).

---

## Schema principles

1. **Messages are append-mostly.** Prefer insert + indexed reads over frequent in-place edits.
2. **Stable IDs.** Use UUIDv7 / ULID / snowflake-style IDs consistently; document the generator owner (`service` or DB).
3. **Conversation-scoped ordering.** Message order is defined by `(conversation_id, seq)` or `(conversation_id, created_at, id)` — pick one and never mix in clients.
4. **Soft delete only when product needs it.** If used, every query must filter deleted rows intentionally.
5. **No unbounded `SELECT *` in hot paths.** Select columns the use-case needs.

### Naming

| Object | Convention |
|--------|------------|
| Tables | `snake_case`, plural nouns (`messages`, `conversations`, `conversation_members`) |
| Columns | `snake_case` (`created_at`, `sender_id`) |
| PK | `id` unless composite is intentional |
| FK | `<entity>_id` |
| Indexes | `idx_<table>_<cols>` |
| Unique | `uq_<table>_<cols>` |

---

## Query patterns

### Cursor pagination (messages)

Prefer keyset/cursor pagination over `OFFSET` for timelines:

```text
WHERE conversation_id = $1
  AND (seq, id) < ($2, $3)   -- or > for forward
ORDER BY seq DESC, id DESC
LIMIT $4
```

API returns an opaque cursor derived from the last row, not raw SQL offsets.

### Batch over N+1

- Load membership, users, and receipts in batches.
- **Message list + reply previews**: after listing messages, collect distinct non-null `reply_to_message_id` values and load with `FindByIDs` / `id = ANY($1)` — never per-row `FindByID` in the list path.
- For gateway fan-out, resolve online targets via Redis/presence service in bulk, not per recipient query in a loop without pipelining.

### Messages table (landed)

| Column | Notes |
|--------|-------|
| `id` | UUID, app-assigned |
| `conversation_id`, `sender_id` | FKs |
| `body` | plain text (emoji are characters in body) |
| `client_msg_id` | unique with `sender_id` for idempotent send |
| `seq` | per-conversation monotonic; allocated with `conversations.next_seq` in the insert transaction |
| `created_at` | timestamptz |
| `reply_to_message_id` | **nullable** UUID FK → `messages(id)` **ON DELETE SET NULL** |

Partial index: `idx_messages_reply_to` on `reply_to_message_id` where not null.

**Do not** encode reply/quote as a body text prefix. Persist the FK; API/WS embed a truncated preview object (see realtime-messaging scenario `message.reply_to`).

### Conversation list head + self unread (landed)

| Store | Columns / rule |
|-------|----------------|
| `conversations` | `last_message_at`, `last_message_seq`, `last_message_preview` (≤120 runes), `last_message_sender_id` — updated in the **same tx** as message insert |
| `conversation_members` | `last_read_seq` (default 0) |
| Unread | Count messages with `seq > last_read_seq` **and** `sender_id <> viewer` — not bare seq diff |
| Mark read | `POST /v1/conversations/{id}/read`; open-room calls it; clamp seq to head |
| Self send | `GREATEST(last_read_seq, msg.seq)` for sender in insert tx |
| List order | `ORDER BY COALESCE(last_message_at, updated_at) DESC` |
| Out of scope | Peer read receipts / read WS events |

List DTO includes `last_message`, `unread_count`, optional `member_count` + `last_message.sender_email` for group preview prefixes, and `members` (batch-loaded) so the sidebar can title DMs by peer email / groups as untitled group.

### Friend requests + friendships (landed)

| Store | Columns / rule |
|-------|----------------|
| `friend_requests` | directed `from_user_id` → `to_user_id`; status `pending` / `accepted` / `rejected`; `from_user_id <> to_user_id` |
| Pending uniqueness | partial unique index on `(from_user_id, to_user_id) WHERE status = 'pending'` |
| Service rule | also reject reverse pending for the same pair (either side already asked) |
| Reject / re-request | reject leaves no friendship; a later request may insert a new pending row |
| `friendships` | undirected edge with canonical `user_a_id < user_b_id` (UUID string order); PK `(user_a_id, user_b_id)` |
| Accept | same tx: mark request accepted + insert friendship (`ON CONFLICT DO NOTHING`) + clear reverse pending if any |

Discovery is email-only via `users.email`. Delete-friend / block / notes are out of scope for the relation MVP.

### Transactions

Use transactions for multi-table writes that must commit together, e.g.:

- create conversation + owner membership
- insert message + bump `next_seq` + **update conversation last_*** + **advance sender last_read_seq**
- accept friend request + insert `friendships` edge

Keep transactions short. Do **not** hold a DB transaction open while waiting on Redis, MQ, or WS write.

```text
// good sketch
tx := begin
alloc seq + touch conversation
insert message (incl. reply_to_message_id)
update conversation last_message_* 
advance sender last_read_seq
commit
publish / hub push          // after commit
```

If the process crashes after commit but before publish, recovery is **outbox** or **CDC**, not “expand the transaction across the bus”.

---

## Migrations

1. Every schema change is a numbered migration pair (up/down or goose sequential).
2. Migrations are forward-compatible with running binaries when possible (expand/contract).
3. Do not edit applied migrations on shared branches; add a new one.
4. Seed data for local dev lives separately from schema migrations.

```bash
# example once tooling exists
migrate -path backend/migrations -database "$DATABASE_URL" up
```

---

## Repository layer rules

- `internal/repo` maps rows ↔ domain types.
- Return `domain` errors for not-found / conflict where meaningful (`ErrNotFound`), wrap driver errors otherwise.
- Context is the first argument: `func (r *MessageRepo) List(ctx context.Context, ...)`.
- Honor `ctx` cancellation on queries (`QueryContext` / pgx equivalents).

---

## Redis usage boundaries

| Allowed | Forbidden as source of truth |
|---------|------------------------------|
| Online presence, conn index | Full message history |
| Rate limits, idempotency keys | Conversation ACL alone |
| Unread counters (with rebuild) | User passwords / tokens long-term without policy |
| Short-lived WS ticket / session meta | |

Every Redis key family needs: key pattern, TTL policy, and a rebuild story if cache is flushed.

---

## Anti-patterns

- Writing messages only to Redis “for speed” without durable store.
- Using `OFFSET` deep pagination on large conversation histories.
- Catch-all repositories that expose raw `*sql.DB` to handlers.
- Mixing MySQL and PostgreSQL SQL dialects in the same file without build tags or separate drivers.
- Updating “last message” in conversation without transactional pairing to the insert when consistency matters for UI list ordering.

---

## Common mistakes (IM-specific)

1. **Clock ordering** — never trust client `created_at` as the sole order key.
2. **Missing member check** — every read/write must enforce conversation membership (or channel ACL) in service/repo, not only in the gateway.
3. **Fan-out write amplification** — do not insert N per-user inbox rows synchronously on the request path without a deliberate model (inbox table vs. shared timeline).
4. **Silent migration drift** — gateway and API binaries on different schema assumptions.

---

## Verification

```bash
# once implemented
cd backend && go test ./internal/repo/...
# migration dry-run in CI against ephemeral Postgres
```
