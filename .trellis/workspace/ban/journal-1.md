# Journal - ban (Part 1)

> AI development session journal
> Started: 2026-07-28

---



## Session 1: Bootstrap easy-im Trellis specs

**Date**: 2026-07-28
**Task**: Bootstrap easy-im Trellis specs
**Branch**: `main`

### Summary

Greenfield repo: git init + replace empty Trellis templates with Go/React IM bootstrap specs; archive 00-bootstrap-guidelines.

### Main Changes

- Wrote backend specs (directory, database, realtime/MQ, errors, logging, quality)
- Wrote frontend specs (directory, components, hooks, state, types, quality)
- Rewrote guides for IM cross-layer/reuse; removed Trellis-product-only content
- git init on main; root .gitignore; archived 00-bootstrap-guidelines

### Git Commits

| Hash | Message |
|------|---------|
| `3d7f0a3` | (see git log) |

### Status

[OK] **Completed**

### Next Steps

- Scaffold backend/ (Go module) and frontend/ (Vite React+TS) monorepo roots
- Re-run trellis-spec-bootstrap after real code lands to replace bootstrap assumptions with source-backed rules


## Session 2: Feature map survey (empty product)

**Date**: 2026-07-28
**Task**: Feature map survey (empty product)
**Branch**: `main`

### Summary

Surveyed easy-im for user-perceptible features; 0 implemented (no backend/frontend). Wrote research/index + grouped feature files, risk/newbie sections, and scaffolding next steps. Archived task.

### Main Changes

- Created task 07-28-feature-map-survey with prd/design/implement
- research/: method, index, 5 feature groups, gaps-and-next, non-product appendix
- Recorded 0 implemented features with evidence; planned_only only from specs

### Git Commits

| Hash | Message |
|------|---------|
| `ba2696d` | (see git log) |

### Status

[OK] **Completed**

### Next Steps

- Scaffold backend/ (Go) and frontend/ (Vite React+TS)
- Re-run feature map after first user-facing slice lands


## Session 3: Monorepo scaffold backend+frontend

**Date**: 2026-07-28
**Task**: Monorepo scaffold backend+frontend
**Branch**: `main`

### Summary

Scaffolded easy-im monorepo: Go API with /healthz, Vite React-TS shell (home + health probe), root README, spec bootstrap notes, and check fixes. Archived monorepo-scaffold.

### Main Changes

- backend/: go module, cmd/api, config, handler healthz + tests, migrations placeholder
- frontend/: Vite React-TS, app routes, api client, realtime placeholder, shared layout
- Root README + directory-structure bootstrap status updates
- Check: healthz encode handling + shared hooks/lib .gitkeep

### Git Commits

| Hash | Message |
|------|---------|
| `6b1fdd7` | (see git log) |
| `ae3de24` | (see git log) |
| `706344f` | (see git log) |
| `0302230` | (see git log) |
| `7c84abb` | (see git log) |

### Testing

- [OK] go test ./...; go build ./cmd/api; curl /healthz; npm run build

### Status

[OK] **Completed**

### Next Steps

- Auth minimal slice or conversation CRUD
- Re-run feature map after first user-facing feature


## Session 4: Feature development roadmap

**Date**: 2026-07-28
**Task**: Feature development roadmap
**Branch**: `main`

### Summary

Documented easy-im phased roadmap from feature map + scaffold: P0–P6, M0–M5, T1–T6 splits, risks, default next T1/T2. Archived feature-dev-roadmap.

### Main Changes

- research/index.md + roadmap.md with map calibration after M0 scaffold
- Suggested Trellis sequence T1 DB/errors → T2 auth → T3 conv → T4 HTTP msg → T5 WS → T6 remap

### Git Commits

| Hash | Message |
|------|---------|
| `2772103` | (see git log) |

### Status

[OK] **Completed**

### Next Steps

- Start T1: Postgres migrations + API error middleware
- Or T2: register/login + /me + login page (M1)


## Session 5: T1 Postgres migrations and API errors

**Date**: 2026-07-28
**Task**: T1 Postgres migrations and API errors
**Branch**: `main`

### Summary

Landed T1/P0: pgx pool, goose migrations with users table, apperr + request_id error JSON, /readyz, compose on 5433. Archived p0-db-errors. Ready for serial T2 auth.

### Main Changes

- backend: apperr, db pool, httpx middleware, migrate cmd, users migration
- docker-compose Postgres 16 on host 5433; README runbooks
- spec bootstrap notes for database and error-handling

### Git Commits

| Hash | Message |
|------|---------|
| `26389d9` | (see git log) |
| `1bb5224` | (see git log) |
| `62c05c2` | (see git log) |
| `81cf1e2` | (see git log) |

### Testing

- [OK] go test ./...; migrate up; healthz/readyz smoke

### Status

[OK] **Completed**

### Next Steps

- T2: register/login + /me + frontend login (M1)


## Session 6: T2 auth M1 and minimalism UI

**Date**: 2026-07-28
**Task**: T2 auth M1 and minimalism UI
**Branch**: `main`

### Summary

Shipped M1 auth (JWT register/login/me + FE session/app shell) and redesigned frontend shell to Minimalism: quiet hierarchy, B/W/gray tokens, restrained interaction. Archived t2-auth-login.

### Main Changes

- Backend auth: bcrypt, JWT, /v1/auth/*, /v1/me, repo/service layers
- Frontend auth pages, Session, protected /app
- Minimalism redesign of home/auth/health/workspace chrome

### Git Commits

| Hash | Message |
|------|---------|
| `ae92e8d` | (see git log) |
| `e715470` | (see git log) |
| `5ffb70e` | (see git log) |
| `4e5be8d` | (see git log) |
| `9fa88f5` | (see git log) |

### Testing

- [OK] go test ./...; npm run build; register/login/me smoke

### Status

[OK] **Completed**

### Next Steps

- T3: conversations create/list + ACL (M2)


## Session 7: T3 conversations M2

**Date**: 2026-07-28
**Task**: T3 conversations M2
**Branch**: `main`

### Summary

M2: conversation create/list/get with membership ACL and workspace UI. Archived t3-conversations.

### Main Changes

- migrations conversations + members; repo/service/handlers
- FE workspace sidebar create/list and empty room

### Git Commits

| Hash | Message |
|------|---------|
| `77fc3d4` | (see git log) |

### Testing

- [OK] go test; npm build; two-user smoke ACL 404

### Status

[OK] **Completed**

### Next Steps

- T4 HTTP messages + history + client_msg_id (M3)


## Session 8: T4 HTTP messages M3

**Date**: 2026-07-28
**Task**: T4 HTTP messages M3
**Branch**: `main`

### Summary

M3: HTTP message send/history with idempotent client_msg_id and FE composer/poll. Next T5 WS.

### Main Changes

- messages table, MessageService, APIs
- ConversationRoom composer + 4s poll

### Git Commits

| Hash | Message |
|------|---------|
| `b26a991
97db6ca chore(task): archive 07-28-t4-http-messages
b26a991 feat: HTTP messages with seq and client_msg_id idempotency (M3)
bf8705c chore: record journal` | (see git log) |

### Testing

- [OK] go test; npm build; idempotent smoke

### Status

[OK] **Completed**

### Next Steps

- T5 single-node WS gateway (M4)


## Session 9: T5 websocket realtime M4

**Date**: 2026-07-28
**Task**: T5 websocket realtime M4
**Branch**: `main`

### Summary

M4: single-node WS push message.created; FE realtime merge. Next optional P5 polish and feature map refresh.

### Main Changes

- hub + /v1/ws + broadcast on send
- frontend connectRealtime

### Git Commits

| Hash | Message |
|------|---------|
| `babd85c` | (see git log) |

### Testing

- [OK] go test; npm build; WS_OK smoke

### Status

[OK] **Completed**

### Next Steps

- T6 refresh feature map; optional P5 receipts/settings


## Session 10: Roadmap mainline complete M0-M4

**Date**: 2026-07-28
**Task**: Roadmap mainline complete M0-M4
**Branch**: `main`

### Summary

Completed Trellis roadmap mainline T1-T6 / M0-M4 with verification. Feature map refreshed. P5/P6 optional remaining.

### Main Changes

- T3 conversations M2, T4 messages M3, T5 WS M4, T6 map

### Git Commits

| Hash | Message |
|------|---------|
| `d87a97b` | (see git log) |

### Testing

- [OK] go test; npm build; FINAL_OK health/me/msg/list; WS_OK earlier

### Status

[OK] **Completed**

### Next Steps

- Optional P5 receipts/settings or production hardening


## Session 11: Frontend i18n en/zh-CN

**Date**: 2026-07-28
**Task**: Frontend i18n en/zh-CN
**Branch**: `main`

### Summary

Added English and Simplified Chinese i18n to the React SPA with i18next, Header language switcher, browser-aware defaults, and frontend spec conventions. typecheck/build green.

### Git Commits

| Hash | Message |
|------|---------|
| `974638b` | (see git log) |
| `9930c01` | (see git log) |
| `613edc7` | (see git log) |

### Status

[OK] **Completed**


## Session 12: WeChat-like chat UI: layout, bubbles, emoji, reply

**Date**: 2026-07-28
**Task**: WeChat-like chat UI: layout, bubbles, emoji, reply
**Branch**: `main`

### Summary

Shipped mature text chat UX: backend reply_to_message_id end-to-end (migration, DTO, WS parity, tests); frontend features/chat with three-band room, greyscale bubbles, optimistic send, emoji toolbar, structured quote/reply; DM title uses peer name and hides per-bubble labels (group only); specs updated for chat ownership, bubble DOM, and message.reply_to code-spec.

### Git Commits

| Hash | Message |
|------|---------|
| `e8f869b` | (see git log) |
| `91667b0` | (see git log) |

### Status

[OK] **Completed**


## Session 13: Conversation list preview and self-only unread

**Date**: 2026-07-28
**Task**: Conversation list preview and self-only unread
**Branch**: `main`

### Summary

Shipped sidebar WeChat-surface list: denormalized conversation last_message on send, members.last_read_seq, peer-only unread_count, POST mark-read on room open, AppShell message.created list patch; group preview sender prefix; specs for list head/unread. Also earlier this session: wechat-chat-ui (bubbles/reply_to) archived separately.

### Git Commits

| Hash | Message |
|------|---------|
| `8f50d35` | (see git log) |
| `2949dde` | (see git log) |

### Status

[OK] **Completed**


## Session 14: 好友关系 MVP（请求/同意/列表）

**Date**: 2026-07-29
**Task**: 好友关系 MVP（请求/同意/列表）
**Branch**: `main`

### Summary

落地 email 好友请求→同意/拒绝与 friendships 无向边；/v1/friends API + 最小 Friends 页；spec 记录 schema 与页面状态；父任务 friends-chat 与子任务 open-chat 规划保留。未改 member_emails 建会话。

### Git Commits

| Hash | Message |
|------|---------|
| `bce98da` | (see git log) |
| `1cd9637` | (see git log) |
| `4149fe4` | (see git log) |

### Status

[OK] **Completed**


## Session 15: 好友驱动开聊（替代邮箱建会话）

**Date**: 2026-07-29
**Task**: 好友驱动开聊（替代邮箱建会话）
**Branch**: `main`

### Summary

好友 get-or-create 1:1：POST /v1/friends/{id}/conversation；移除 member_emails 建会话与 AppShell 邮箱表单；Friends 页 Message 进房；FindDirectBetween 按 last_message_at/created_at 复用；spec 记录 OpenDirect；历史会话发消息仍不校验好友。

### Git Commits

| Hash | Message |
|------|---------|
| `88c4084` | (see git log) |
| `2ef1a01` | (see git log) |
| `1fcd66e` | (see git log) |

### Status

[OK] **Completed**


## Session 16: P5.a Read receipts + P5.b Typing indicators

**Date**: 2026-07-29
**Task**: P5.a Read receipts + P5.b Typing indicators
**Branch**: `main`

### Summary

Implemented bidirectional WebSocket protocol (hub inbound frame parsing + FrameHandler), read receipt broadcasts (message.read via MarkRead), typing indicators (typing.start/stop with 3s server timeout, 4s client timeout), and frontend UI (gray checkmarks, animated dots). Updated realtime-messaging and frontend specs.

### Git Commits

| Hash | Message |
|------|---------|
| `39141d1` | (see git log) |
| `7fd2a77` | (see git log) |

### Status

[OK] **Completed**


## Session 17: P5.c Online presence (presence dots)

**Date**: 2026-07-31
**Task**: P5.c Online presence (presence dots)
**Branch**: `main`

### Summary

Implemented online/offline presence dots: hub IsOnline/OnlineUserIDs + PresenceBroadcaster on 0-1 transitions, friend-scoped presence.changed broadcasts via ListFriendIDs, online field on /v1/friends and conversation members. Refactored frontend to a single app-wide WebSocket (RealtimeProvider/useRealtime), fixing the two-socket activeWs race. Presence dots in friends list and DM header.

### Git Commits

| Hash | Message |
|------|---------|
| `a31ce78` | (see git log) |
| `a92cd42` | (see git log) |

### Status

[OK] **Completed**


## Session 18: P5.d Settings page (profile, display name, password)

**Date**: 2026-07-31
**Task**: P5.d Settings page (profile, display name, password)
**Branch**: `main`

### Summary

Implemented full settings page: users.display_name migration, AuthService UpdateDisplayName (max 64 runes) + ChangePassword (bcrypt verify current), UserRepo FindRecordByID/UpdateDisplayName/UpdatePassword with display_name in all user selects, PATCH /v1/me/profile + POST /v1/me/password, /v1/me returns profile with created_at, CORS allows PATCH. Frontend: /settings page with profile/display-name/password forms, Session refreshUser, chat UI prefers display_name over email short-name. Tokens kept valid after password change.

### Git Commits

| Hash | Message |
|------|---------|
| `a08521c` | (see git log) |
| `72e10fc` | (see git log) |

### Status

[OK] **Completed**


## Session 19: P5.e Message edit and recall

**Date**: 2026-07-31
**Task**: P5.e Message edit and recall
**Branch**: `main`

### Summary

Implemented message edit and recall: messages.edited_at/recalled_at migration, MessageRepo UpdateBody/MarkRecalled with conditional head-preview refresh, MessageService Edit/Recall (own-message-only, 5-min window, no double recall), PATCH messages/{id} + POST messages/{id}/recall, message.edited/message.recalled WS events with full DTO. Frontend: bubble edit mode, recall button, recalled placeholder (no hard delete), edited marker, conversation-list preview updates. Completes all five P5 items.

### Git Commits

| Hash | Message |
|------|---------|
| `1bea908` | (see git log) |
| `beea97d` | (see git log) |

### Status

[OK] **Completed**


## Session 20: P6 Production CORS and auth hardening

**Date**: 2026-07-31
**Task**: P6 Production CORS and auth hardening
**Branch**: `main`

### Summary

Migrated auth to HttpOnly SameSite=Lax cookie sessions (JWT in cookie, Secure in prod), added POST /v1/auth/logout, WS auth via cookie with CheckOrigin allowlist validation, env-driven CORS allowlist with credentials + Vary, empty-secret startup abort, TTL 24h. Frontend: removed localStorage token and all API token params, credentials: include, Session tracks user only, realtime connects without ?token=. Fixed realtime/index.ts -> .tsx (JSX in .ts).

### Git Commits

| Hash | Message |
|------|---------|
| `0061cf2` | (see git log) |
| `f522deb` | (see git log) |

### Status

[OK] **Completed**


## Session 21: P6 Offline push (Web Push PWA)

**Date**: 2026-07-31
**Task**: P6 Offline push (Web Push PWA)
**Branch**: `main`

### Summary

Added offline Web Push delivery. Backend: push_subscriptions table + POST/DELETE /v1/push/subscriptions + GET /v1/push/vapid; internal/push (VAPID + aes128gcm send via wuc656/webpush-go, session aggregator, offline handler, flusher with 410/404 prune); internal/mq (franz-go producer/consumer, im.messages + im.presence topics); cmd/worker consumes Kafka, learns online set from presence, pushes only offline members; MessageService.Send produces post-durable-write; hub presence events to Kafka. Frontend: public/sw.js + manifest + icons, /settings push toggle (permission → subscribe → register). go.mod upgraded 1.22 → 1.25.2 (webpush-go requirement). docker-compose gains KRaft Kafka (host 19092). E2E verified: offline push delivered (mock 201 + valid VAPID sig), stale sub pruned on 410, online members skipped.

### Git Commits

| Hash | Message |
|------|---------|
| `a1170b7` | feat(push): offline Web Push delivery via Kafka worker |
| `ea4d4b1` | feat(pwa): service worker, manifest, and push settings toggle |
| `4ee6b57` | docs(spec): document Kafka offline-push bus and push_subscriptions |
| `a070e58` | chore(task): record 07-31-p6-offline-push artifacts |

### Status

[OK] **Completed**
