# Design: conv list preview + unread

## Architecture

```text
Send message (existing)
  → MessageRepo.Insert tx: alloc seq, insert row, **update conversation last_*** 
  → bump sender member last_read_seq ≥ msg.seq
  → hub message.created (unchanged shape + clients patch list)

GET /v1/conversations
  → rows: conversation + last_* + member.last_read_seq
  → unread_count: count peer msgs seq > last_read  (batch, not N+1 per row in loop without care)

POST /v1/conversations/{id}/read
  → set last_read_seq = max(current, requested|head_seq)

AppShell
  → list state holds preview/unread
  → connectRealtime once: on message.created patch row + reorder
  → open room: ConversationRoom load ok → markRead(id)
```

## Schema

Migration (new goose file):

```sql
-- conversations head
ALTER TABLE conversations
  ADD COLUMN last_message_at TIMESTAMPTZ NULL,
  ADD COLUMN last_message_seq BIGINT NULL,
  ADD COLUMN last_message_preview TEXT NULL,
  ADD COLUMN last_message_sender_id UUID NULL REFERENCES users (id);

-- members read cursor
ALTER TABLE conversation_members
  ADD COLUMN last_read_seq BIGINT NOT NULL DEFAULT 0;
```

Down: drop columns.

**Preview truncate**: service-side rune cap **120** (align reply preview habit) on store to `last_message_preview`.

## Domain / API

### List item DTO (extends conversation)

```json
{
  "id": "...",
  "title": null,
  "created_by": "...",
  "created_at": "...",
  "updated_at": "...",
  "last_message": {
    "seq": 12,
    "body": "truncated preview",
    "sender_id": "...",
    "created_at": "RFC3339"
  },
  "unread_count": 3
}
```

`last_message` may be `null` if never sent.

### Mark read

`POST /v1/conversations/{id}/read`

```json
{ "seq": 12 }
```

`seq` optional; default = conversation `last_message_seq` or 0.  
Response: `{ "last_read_seq": 12, "unread_count": 0 }` or 204 + client local clear — prefer small JSON for FE.

CORS: allow POST already; no new methods required. If PATCH desired, extend CORS — **prefer POST** to avoid CORS churn.

## Unread computation

**Semantic**: count messages in conv where `seq > last_read_seq AND sender_id <> viewer`.

**List strategy (MVP)**:

1. Load membership conversations with last_* and `m.last_read_seq` in one query.
2. For unread: either
   - **Preferred for correctness at small scale**: one grouped query  
     `SELECT conversation_id, COUNT(*) FROM messages WHERE conversation_id = ANY($1) AND seq > …` hard per-row — better:  
     `SELECT conversation_id, COUNT(*) FROM messages msg JOIN …` with `(conversation_id, last_read_seq)` unnest; or per-conv count in SQL lateral.
   - **Optional denorm later**: `members.unread_count` maintained on send/read — out of MVP unless count is slow.

Do **not** use bare `last_seq - last_read_seq` as the product number.

## Send path changes

In `MessageRepo.Insert` transaction after message insert (same tx that bumps `next_seq`):

```sql
UPDATE conversations SET
  last_message_at = $created,
  last_message_seq = $seq,
  last_message_preview = $preview,
  last_message_sender_id = $sender,
  updated_at = now()
WHERE id = $conv;
```

After commit (or in service after Insert returns):  
`UPDATE conversation_members SET last_read_seq = GREATEST(last_read_seq, $seq) WHERE conversation_id=$c AND user_id=$sender`.

If GREATEST in same tx as insert: OK and preferred (sender never sees self-unread).

## Mark read path

- ACL: must be member.
- `last_read_seq = GREATEST(last_read_seq, seq)`.
- `seq` must be ≥ 0; if `seq` > head, clamp to head (invalid high → clamp, not error) **or** reject — **recommend clamp** for client clock-skew simplicity.

## Frontend

### Types

Extend `api/conversations.ts`; add `markConversationRead(token, id, seq?)`.

### AppShell

- Hold `items: ConversationListItem[]`.
- On mount: list + **workspace-level** `connectRealtime`:
  - find item by `conversation_id`
  - set last_message from payload body/sender/seq/created_at
  - if `sender_id !== me` and `!useMatch('/app/c/'+id)` → unread_count += 1
  - if active room for that id → do not bump unread (markRead will run / already caught up)
  - sort by last_message.created_at || updated_at desc
- Room open: after messages load success, `markConversationRead` then set that item unread_count=0 and last_read locally if tracked.

### Row UI

- Title: existing untitled / title; optional peer name if members present on list — **list today may omit members**.  
  - Design choice: list endpoint **does not require full members** for MVP; DM title stays title or untitled unless we add peer join later.  
  - **Enhancement in-scope if cheap**: list returns no members; FE keeps showing title; peer-name-as-title was room-only in wechat-chat-ui. Sidebar may still show untitled for DM without title — acceptable MVP **or** lightweight: include other member email on list via join. **Recommend**: list query left join for display name of «the other member» when member_count=2 — optional polish in implement if time; not blocking AC if title empty shows untitled.
- Preview line + time (`Intl.RelativeTimeFormat` or short time).
- Badge if unread_count > 0.

### i18n

`chat.youPrefix`, `workspace.unreadAria`, etc.

## Compatibility

- Old rows: last_* null, unread 0.
- WS payload unchanged; FE derives list patch from message DTO.
- reply_to ignored for preview (use `body` only).

## Trade-offs

| Topic | Choice | Why |
|-------|--------|-----|
| Denorm last_* | yes | list read path |
| Peer read WS | no | grill A |
| unread denorm column | no for MVP | count OK at small N |
| POST vs PATCH read | POST | CORS |

## Risks

- AppShell + Room both connecting WS → **one shell-level connection** preferred; room keeps its own today — dual conn per user is already possible (multi-tab). Same tab dual `connectRealtime` from shell+room = two sockets. **Mitigate**: either lift WS to shell and pass messages down, or accept 2 conns in MVP (hub supports multi device). **Recommend MVP accept 2 conns** to avoid large room refactor; document follow-up to unify.

- Unread +1 on WS while markRead in flight → possible flicker; markRead response wins.

## Rollback

goose down; FE ignore new fields.
