# Implement: conv list preview + unread

## Ordered checklist

### Backend

1. [ ] Migration: conversations last_* columns; members.last_read_seq.
2. [ ] Domain Conversation (+ list view fields) / member read cursor helpers.
3. [ ] MessageRepo.Insert: update conversation head in same tx; advance sender last_read_seq.
4. [ ] ConversationRepo.ListForUser: select last_*; order by COALESCE(last_message_at, updated_at) DESC.
5. [ ] Unread counts for list (batch/grouped query by peer messages).
6. [ ] Service+handler: extend list DTO with `last_message` + `unread_count`.
7. [ ] MarkRead service + `POST /v1/conversations/{id}/read` route.
8. [ ] Tests: send updates head; list unread peer-only; mark read clears; self-send no self-unread.

### Frontend

9. [ ] `api/conversations.ts` types + `markConversationRead`.
10. [ ] AppShell: render preview, time, badge; sort client-side if needed.
11. [ ] AppShell: workspace `connectRealtime` patch list on `message.created`.
12. [ ] ConversationRoom: after successful load, mark read; optional callback to parent to zero badge (or rely on shared state / re-fetch list).
13. [ ] i18n en/zh-CN for you-prefix / unread.
14. [ ] CSS: badge + preview line (minimal, no green).

### Verify

15. [ ] `go test ./internal/service/ ./internal/handler/ …`
16. [ ] `npm run build`
17. [ ] migrate up; manual two-user: preview, unread, open clear, WS sidebar.

## Validation commands

```bash
cd backend && go test ./internal/service/ ./internal/handler/ -count=1
export DATABASE_URL='postgres://easyim:easyim@localhost:5433/easyim?sslmode=disable'
cd backend && go run ./cmd/migrate up
cd frontend && npm run build
```

## Risky files

| File | Risk |
|------|------|
| `message_repo.go` Insert tx | head update miss / tx length |
| `AppShell.tsx` | dual WS with room; list patch races |
| List unread SQL | N+1 or wrong peer filter |

## jsonl

Curate implement/check with frontend directory/state + backend database/realtime + this design/prd.

## Gate

User approves planning summary → `task.py start` → implement. No product code before start.
