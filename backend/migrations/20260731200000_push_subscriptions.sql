-- +goose Up
-- Web Push subscriptions for offline notification delivery (P6).
-- One row per (user, browser push endpoint). Workers read these to send
-- pushes to users whose connections are all offline.

CREATE TABLE push_subscriptions (
    id         UUID PRIMARY KEY,
    user_id    UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    endpoint   TEXT NOT NULL,
    p256dh     TEXT NOT NULL,
    auth       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX uq_push_subs_user_endpoint ON push_subscriptions (user_id, endpoint);

-- +goose Down
DROP TABLE IF EXISTS push_subscriptions;
