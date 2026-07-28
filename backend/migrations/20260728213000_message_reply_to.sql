-- +goose Up
ALTER TABLE messages
  ADD COLUMN reply_to_message_id UUID NULL
  REFERENCES messages (id) ON DELETE SET NULL;

CREATE INDEX idx_messages_reply_to ON messages (reply_to_message_id)
  WHERE reply_to_message_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_messages_reply_to;
ALTER TABLE messages DROP COLUMN IF EXISTS reply_to_message_id;
