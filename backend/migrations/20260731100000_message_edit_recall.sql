-- +goose Up
ALTER TABLE messages
  ADD COLUMN edited_at  TIMESTAMPTZ NULL,
  ADD COLUMN recalled_at TIMESTAMPTZ NULL;

-- +goose Down
ALTER TABLE messages
  DROP COLUMN edited_at,
  DROP COLUMN recalled_at;
