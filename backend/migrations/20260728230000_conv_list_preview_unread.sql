-- +goose Up
ALTER TABLE conversations
  ADD COLUMN last_message_at TIMESTAMPTZ NULL,
  ADD COLUMN last_message_seq BIGINT NULL,
  ADD COLUMN last_message_preview TEXT NULL,
  ADD COLUMN last_message_sender_id UUID NULL REFERENCES users (id);

ALTER TABLE conversation_members
  ADD COLUMN last_read_seq BIGINT NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE conversation_members DROP COLUMN IF EXISTS last_read_seq;
ALTER TABLE conversations
  DROP COLUMN IF EXISTS last_message_sender_id,
  DROP COLUMN IF EXISTS last_message_preview,
  DROP COLUMN IF EXISTS last_message_seq,
  DROP COLUMN IF EXISTS last_message_at;
