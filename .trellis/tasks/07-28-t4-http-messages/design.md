# Design T4 Messages

## Schema
```sql
messages(
  id UUID PK,
  conversation_id UUID NOT NULL REFERENCES conversations,
  sender_id UUID NOT NULL REFERENCES users,
  body TEXT NOT NULL,
  client_msg_id TEXT NOT NULL,
  seq BIGINT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(sender_id, client_msg_id),
  UNIQUE(conversation_id, seq)
)
-- seq via per-conversation counter on conversations.seq_next or MAX+1 in tx
```

Use conversations.next_seq counter column for atomic seq assignment.

## API
POST body: `{ "body": "...", "client_msg_id": "uuid" }`
GET `?limit=50&before_seq=` keyset older messages

## FE
ConversationRoom: list messages + composer; poll or refresh button for M3 (simple reload on focus/interval optional short poll 5s).
