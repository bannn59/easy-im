-- +goose Up
ALTER TABLE conversations ADD COLUMN next_seq BIGINT NOT NULL DEFAULT 1;

CREATE TABLE messages (
    id              UUID PRIMARY KEY,
    conversation_id UUID NOT NULL REFERENCES conversations (id) ON DELETE CASCADE,
    sender_id       UUID NOT NULL REFERENCES users (id),
    body            TEXT NOT NULL,
    client_msg_id   TEXT NOT NULL,
    seq             BIGINT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_messages_sender_client UNIQUE (sender_id, client_msg_id),
    CONSTRAINT uq_messages_conv_seq UNIQUE (conversation_id, seq)
);

CREATE INDEX idx_messages_conv_seq ON messages (conversation_id, seq DESC);

-- +goose Down
DROP TABLE IF EXISTS messages;
ALTER TABLE conversations DROP COLUMN IF EXISTS next_seq;
