# Design: P5.a Read Receipts + P5.b Typing Indicators

## Architecture Overview

Both features require the same foundational change: **bidirectional WebSocket protocol**. The hub currently only pushes outbound frames (`message.created`); we need it to also parse and dispatch inbound frames from clients.

```
Client  ──inbound frame──>  Hub.ReadPump  ──dispatch──>  handler
                                                           │
                                                           ├── typing events  → Hub.BroadcastToConversation
                                                           └── (read receipts → HTTP-only, no inbound needed)

ConversationService.MarkRead  ──broadcast──>  Hub.PublishToUsers (message.read event)
```

## 1. Hub: Inbound Frame Routing

### Design

Add a `FrameHandler` callback field to `Hub`. `ReadPump` parses each inbound frame as JSON, extracts the `type` field, and calls `FrameHandler(userID, frame)`.

```go
// hub.go
type InboundFrame struct {
    Type    string          `json:"type"`
    Payload json.RawMessage `json:"payload"`
}

type Hub struct {
    mu           sync.RWMutex
    clients      map[string]map[*Client]struct{}
    FrameHandler func(userID string, frame InboundFrame)  // set by ws.go after hub creation
}
```

`ReadPump` changes:
```go
func (c *Client) ReadPump(onClose func()) {
    defer onClose()
    for {
        _, raw, err := c.Conn.ReadMessage()
        if err != nil { return }
        var f InboundFrame
        if err := json.Unmarshal(raw, &f); err != nil { continue }
        if h.FrameHandler != nil {
            h.FrameHandler(c.UserID, f)
        }
    }
}
```

**Why callback instead of interface:** The hub is infrastructure, not business logic. A callback keeps the hub decoupled from domain concerns. The actual routing logic lives in `ws.go` or a thin dispatcher.

### Client needs Hub reference

Currently `Client` has no reference to `Hub`. `ReadPump` is a method on `Client` but needs the hub's `FrameHandler`. Two options:

- **Option A**: Make `ReadPump` a method on Hub that takes a Client → cleaner, Hub already owns the state
- **Option B**: Add `Hub *Hub` field to Client

**Decision: Option A** — `func (h *Hub) ReadPump(c *Client, onClose func())`. This matches the existing `Register`/`Unregister` which are Hub methods.

## 2. Hub: BroadcastToConversation

New method for typing indicators — broadcast to all members of a conversation except the sender:

```go
func (h *Hub) BroadcastToConversation(memberIDs []string, exceptUserID string, event Event)
```

Reuses `PublishToUsers` internally but skips `exceptUserID`. For typing, the caller (dispatcher in `ws.go`) will fetch member IDs via `MembershipChecker.ListMemberIDs` before calling this.

## 3. Hub: Typing Timeout

Ephemeral server-side typing state with auto-expiry:

```go
type Hub struct {
    // ...existing fields...
    typingMu     sync.Mutex
    typingTimers map[string]*time.AfterFunc  // key: "conversationID:userID"
}
```

On `typing.start`: set/reset a 3-second `time.AfterFunc` that broadcasts `typing.stopped` on expiry.
On `typing.stop`: cancel the timer, broadcast `typing.stopped`.

Key format `"conversationID:userID"` allows a user to be typing in multiple conversations simultaneously.

## 4. Read Receipts: Broadcast from ConversationService

### Design

Add `RealtimePublisher` to `ConversationService` (same pattern as `MessageService`):

```go
type ConversationService struct {
    convs  ConversationRepo
    users  UserLookup
    friends FriendshipChecker
    rt     RealtimePublisher  // NEW — optional, nil-safe
}
```

After `MarkRead` succeeds, broadcast `message.read` to all other conversation members:

```go
event payload: {
    "conversation_id": "...",
    "reader_id": "...",
    "last_read_seq": 42
}
```

Event type: `"message.read"`.

**No inbound WS frame needed** — read receipts piggyback on the existing `POST /v1/conversations/{id}/read` HTTP endpoint. This is simpler and matches the current architecture where `markConversationRead` is called from the frontend via HTTP.

### Frontend read receipt state

`ConversationRoom` tracks `peerReadSeq: number` (or `Map<string, number>` for group chats) updated from `message.read` WebSocket events. `MessageBubble` checks `msg.seq <= peerReadSeq` to render ✓ vs ✓✓.

## 5. Typing Indicators: Frame Dispatch in ws.go

`ws.go` sets `hub.FrameHandler` to a function that:
1. Validates the frame `type` (`typing.start` / `typing.stop`)
2. Extracts `conversation_id` from payload
3. Verifies the user is a member of that conversation
4. Calls `hub.BroadcastToConversation(memberIDs, senderID, event)`
5. Manages typing timer in hub

## 6. Frontend: Realtime Client Extensions

### New event handlers

```typescript
type RealtimeHandlers = {
    onMessageCreated?: (m: Message) => void;
    onMessageRead?: (data: { conversation_id: string; reader_id: string; last_read_seq: number }) => void;
    onTypingStarted?: (data: { conversation_id: string; user_id: string }) => void;
    onTypingStopped?: (data: { conversation_id: string; user_id: string }) => void;
    onStatus?: (s: RealtimeStatus) => void;
};
```

### New send function

```typescript
export function sendFrame(ws: WebSocket | null, type: string, payload: unknown): void
```

Exposed for `ConversationRoom` to send `typing.start` / `typing.stop` frames. The realtime module holds a reference to the active WebSocket.

### Typing debounce

Client-side: when user types in Composer, emit `typing.start` (debounced: skip if emitted within last 2s). On composer blur/clear or 4s inactivity, emit `typing.stop`.

## 7. Frontend: ChatItem Extension

```typescript
type ChatItem = Message & {
    status: 'pending' | 'sent' | 'failed';
    localKey: string;
    isRead?: boolean;  // NEW — true when all peers have read this message
};
```

## 8. Data Flow Summary

### Read Receipt Flow
```
User B opens conversation
  → POST /v1/conversations/{id}/read  (HTTP)
  → ConversationService.MarkRead
  → conv_repo.MarkRead (DB update)
  → broadcast: hub.PublishToUsers(otherMemberIDs, "message.read", {conversation_id, reader_id, last_read_seq})
  → User A's frontend receives event
  → Updates peerReadSeq state
  → MessageBubbles re-render: ✓ → ✓✓ for seq <= peerReadSeq
```

### Typing Indicator Flow
```
User A types in Composer
  → Client sends WS frame: { type: "typing.start", payload: { conversation_id } }
  → Hub.ReadPump parses frame
  → FrameHandler dispatches to typing handler
  → Verify A is member of conversation
  → Hub.BroadcastToConversation(memberIDs, exceptA, "typing.started", {conversation_id, user_id})
  → Reset 3-second server timeout
  → User B's frontend receives event
  → Shows "A 正在输入..." near composer
  → 4-second client timeout hides indicator if no further events

User A stops typing for 3 seconds
  → Server timer fires
  → Hub broadcasts "typing.stopped" to other members
  → User B's frontend hides indicator
```
