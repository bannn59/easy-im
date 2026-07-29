# Implement: P5.a Read Receipts + P5.b Typing Indicators

## Implementation Order

Work bottom-up: hub layer → service layer → handler layer → frontend realtime → frontend UI.

---

### Step 1: Hub — Inbound frame parsing

**Files:** `backend/internal/hub/hub.go`

- [x] Add `InboundFrame` struct (Type + Payload)
- [x] Add `FrameHandler func(userID string, frame InboundFrame)` field to Hub
- [x] Add `typingMu sync.Mutex` and `typingTimers map[string]*time.Timer` to Hub
- [x] Change `ReadPump` from Client method to Hub method: `func (h *Hub) ReadPump(c *Client, onClose func())`
- [x] In ReadPump: parse JSON, extract type, call FrameHandler
- [x] Add `BroadcastToConversation(memberIDs []string, exceptUserID string, event Event)` method
- [x] Add `SetTyping(conversationID, userID string, onTypingStop func())` and `ClearTyping(conversationID, userID string)` methods for timer management

**Verify:** Compile passes. Existing tests still pass. ✅

### Step 2: Hub — Update callers of ReadPump

**Files:** `backend/internal/handler/ws.go`

- [x] Update `ws.go` to call `h.Hub.ReadPump(client, onClose)` instead of `client.ReadPump(onClose)`

**Verify:** Compile passes. WS connections still work. ✅

### Step 3: ConversationService — RealtimePublisher for read receipts

**Files:** `backend/internal/service/conversation_service.go`

- [x] Add `rt RealtimePublisher` field to ConversationService (same interface as MessageService)
- [x] Update constructor/New function to accept optional RealtimePublisher
- [x] In `MarkRead`, after successful repo call, broadcast `message.read` event to other members
- [x] Add `ListMemberIDs` to `ConversationStore` interface

**Verify:** Compile passes. MarkRead HTTP endpoint still works. ✅

### Step 4: Wiring — Connect RealtimePublisher to ConversationService

**Files:** `backend/internal/app/api.go`

- [x] Pass hub (as RealtimePublisher) to ConversationService constructor
- [x] Pass `members` (ConversationRepo as MembershipChecker) to Deps

**Verify:** Compile passes. ✅

### Step 5: WS Frame Dispatcher — Typing handler

**Files:** `backend/internal/handler/ws.go`

- [x] Set `hub.FrameHandler` to `ws.HandleFrame` in router.go
- [x] Implement `typing.start` handler
- [x] Implement `typing.stop` handler
- [x] Log and ignore unknown frame types

**Verify:** Compile passes. ✅

### Step 6: Frontend — Realtime client extensions

**Files:** `frontend/src/realtime/index.ts`

- [x] Add `onMessageRead`, `onTypingStarted`, `onTypingStopped` to `RealtimeHandlers`
- [x] Add `switch` in `ws.onmessage` for new event types
- [x] Expose `sendFrame(type, payload)` function

**Verify:** TypeScript compiles. ✅

### Step 7: Frontend — Read receipt UI

**Files:** `frontend/src/features/chat/ConversationRoom.tsx`, `MessageBubble.tsx`, `types.ts`

- [x] Add `isRead?: boolean` to `ChatItem` type
- [x] Track `peerReadSeq` state per conversation
- [x] Handle `onMessageRead` event
- [x] Compute `effectiveReadSeq` (min of peers for group, direct for DM)
- [x] Show ✓ / ✓✓ in MessageBubble for own messages

**Verify:** TypeScript compiles. ✅

### Step 8: Frontend — Typing indicator UI

**Files:** `frontend/src/features/chat/ConversationRoom.tsx`

- [x] Track `typingUsers` state
- [x] Handle `onTypingStarted` / `onTypingStopped` events with 4s client timeout
- [x] Render typing indicator between MessageList and Composer
- [x] Send `typing.start` (debounced 2s) and `typing.stop` from Composer integration

**Verify:** TypeScript compiles. ✅

### Step 9: i18n + CSS

**Files:** `en.json`, `zh-CN.json`, `index.css`

- [x] Add translation keys for typing indicator text
- [x] Add CSS for checkmark and typing indicator with animated dots
- [x] Add `prefers-reduced-motion` support for animation

**Verify:** TypeScript compiles. ✅

---

## Risky Files / Rollback Points

| File | Risk | Rollback |
|------|------|----------|
| `hub/hub.go` | ReadPump signature change affects all callers | Revert method change |
| `ws.go` | FrameHandler wiring | Remove FrameHandler assignment |
| `conversation_service.go` | MarkRead broadcast addition | Guard with nil check on `rt` |

## Validation Commands

```bash
# Backend compile
cd backend && go build ./...

# Frontend compile
cd frontend && npx tsc --noEmit

# Manual WS test (typing)
wscat -c "ws://localhost:8080/v1/ws?token=TOKEN"
> {"type":"typing.start","payload":{"conversation_id":"UUID"}}

# Manual HTTP test (read receipt)
curl -X POST http://localhost:8080/v1/conversations/UUID/read \
  -H "Authorization: Bearer TOKEN"
```
