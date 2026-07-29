# P5.a + P5.b: Read Receipts & Typing Indicators

## Goal

Add two real-time interaction signals — read receipts and typing indicators — so the chat feels "alive." Both features reuse the existing hub broadcast primitive and require a new client-to-server WebSocket protocol.

## Background

### Current technical foundation

- **Hub**: single-process in-memory connection registry with `PublishToUsers(userID[], event)`. Only outbound frames exist (`message.created`); `ReadPump` discards all inbound frames.
- **Message model**: append-only, text-only. `conversation_members.last_read_seq` tracks per-user read position.
- **Frontend realtime**: handles only `message.created` event type. `ConversationRoom` already calls `markConversationRead` on each incoming message.
- **DM detection**: no `is_dm` field; inferred at runtime via `MemberCount == 2`.
- **Design language**: minimal black/white/gray.

## Scope (confirmed)

P5.a Read receipts + P5.b Typing indicators. All other P5 items deferred.

## Requirements

### R1 — Client-to-server WebSocket protocol

Hub `ReadPump` must parse inbound frames and route them by `type`. Initial supported inbound types:

| Client frame `type` | Payload | Routing |
|---------------------|---------|---------|
| `typing.start` | `{ conversation_id }` | Relay as `typing.started` to other conversation members |
| `typing.stop` | `{ conversation_id }` | Relay as `typing.stopped` to other conversation members |

Read receipts do NOT need a client-to-server frame — the server infers read state from the existing `POST /v1/conversations/{id}/read` call and broadcasts the event.

### R2 — Read receipts (P5.a)

**Server side:**
- When `POST /v1/conversations/{id}/read` is called, broadcast a `message.read` event to all other conversation members.
- Event payload: `{ conversation_id, reader_id, last_read_seq }`.

**Frontend:**
- Each message bubble (for messages sent by the current user) shows a checkmark indicator:
  - Single gray ✓ = sent but not yet read by the other party (i.e., `msg.seq > peerLastReadSeq`)
  - Double gray ✓✓ = read (i.e., `msg.seq <= peerLastReadSeq`)
- The "peer's last read seq" is tracked per conversation, updated from incoming `message.read` WebSocket events.
- Group chats: show double ✓✓ only when ALL other members have read (i.e., `msg.seq <= min(allPeerLastReadSeqs)`). Show single ✓ otherwise.

### R3 — Typing indicators (P5.b)

**Server side:**
- On receiving `typing.start`, broadcast `typing.started` to other conversation members.
- On receiving `typing.stop`, broadcast `typing.stopped` to other conversation members.
- Server enforces a 3-second timeout: if no `typing.stop` or new `typing.start` arrives within 3s, auto-broadcast `typing.stopped`.
- Typing state is ephemeral — no persistence.

**Frontend:**
- Show a typing indicator in the message list area (near the bottom, above the composer):
  - DM: "对方正在输入..." (or a minimal animated dots indicator)
  - Group: "X 正在输入..." (show the typing user's name)
  - Multiple users typing: "X 和 Y 正在输入..."
- Client sends `typing.start` when the user begins typing in the composer (debounced: only send if not sent in the last 2s).
- Client sends `typing.stop` when the composer is cleared or the user navigates away.
- Hide the indicator on receiving `typing.stopped` or after a 4-second client-side timeout (slightly longer than server's 3s to account for network jitter).

## Acceptance Criteria

### AC1 — Read receipts
- [ ] Sending a message in a DM shows single gray ✓ next to the message
- [ ] When the other user reads the conversation, the sender's messages update to double gray ✓✓
- [ ] In a group chat, ✓✓ appears only when all other members have read
- [ ] Read receipt updates arrive via WebSocket without page refresh
- [ ] Existing `markConversationRead` flow still works (unread counts update)

### AC2 — Typing indicators
- [ ] When user A starts typing in a DM, user B sees a typing indicator near the bottom of the message list
- [ ] The indicator disappears within ~4 seconds after user A stops typing
- [ ] In a group chat, the indicator shows who is typing by name
- [ ] The typing indicator does not persist after navigating away and back
- [ ] Rapid typing does not flood the server (debounce works)

### AC3 — WebSocket protocol
- [ ] Hub correctly parses and routes inbound frames by `type`
- [ ] Unknown frame types are logged and ignored (no crash)
- [ ] Existing `message.created` outbound flow is unaffected
- [ ] Connection drops clean up typing state server-side

## Out of Scope

- Multi-node / Redis presence (P6)
- Offline push notifications (P6)
- Production CORS/JWT hardening (P6)
- Audio/video, full-text search, OAuth/SSO
- P5.c (online presence), P5.d (settings page), P5.e (edit/recall)
- "Delivered" vs "read" distinction (only "sent" and "read" states)
- Read receipt privacy settings (e.g., disable read receipts)

## Open Questions

_(none — all blocking decisions resolved)_
