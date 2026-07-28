# Design: T3 Conversations

## Schema
```sql
conversations(id UUID PK, title TEXT NULL, created_by UUID NOT NULL REFERENCES users, created_at, updated_at)
conversation_members(conversation_id UUID, user_id UUID, joined_at, PRIMARY KEY (conversation_id, user_id))
```

## API
- POST /v1/conversations `{ "member_emails": ["a@b.com"], "title": "optional" }` → 201 conversation+members
- GET /v1/conversations → list for me
- GET /v1/conversations/{id} → detail if member

Creator always member. member_emails resolved via users.email; unknown email → 400.

## Auth
Context key userID from JWT via middleware on conversation routes.

## FE
AppShell layout: sidebar (list + new) + outlet for /app and /app/c/:id empty pane.
