-- +goose Up
-- Initial schema for T2 auth (no auth API in this migration task).
-- Application assigns users.id (UUID text/bytes); no DB default required.

CREATE TABLE users (
    id            UUID PRIMARY KEY,
    email         TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX uq_users_email ON users (email);

-- +goose Down
DROP TABLE IF EXISTS users;
