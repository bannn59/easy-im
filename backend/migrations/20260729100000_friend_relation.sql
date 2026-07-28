-- +goose Up
CREATE TABLE friend_requests (
    id           UUID PRIMARY KEY,
    from_user_id UUID NOT NULL REFERENCES users (id),
    to_user_id   UUID NOT NULL REFERENCES users (id),
    status       TEXT NOT NULL CHECK (status IN ('pending', 'accepted', 'rejected')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    responded_at TIMESTAMPTZ,
    CHECK (from_user_id <> to_user_id)
);

-- At most one active pending request per directed pair.
CREATE UNIQUE INDEX uq_friend_requests_pending
    ON friend_requests (from_user_id, to_user_id)
    WHERE status = 'pending';

CREATE INDEX idx_friend_requests_to_pending
    ON friend_requests (to_user_id, created_at DESC)
    WHERE status = 'pending';

CREATE INDEX idx_friend_requests_from
    ON friend_requests (from_user_id, created_at DESC);

-- Undirected edge; store canonical order user_a_id < user_b_id (UUID text/bytes order).
CREATE TABLE friendships (
    user_a_id  UUID NOT NULL REFERENCES users (id),
    user_b_id  UUID NOT NULL REFERENCES users (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (user_a_id, user_b_id),
    CHECK (user_a_id < user_b_id)
);

CREATE INDEX idx_friendships_user_b ON friendships (user_b_id);

-- +goose Down
DROP TABLE IF EXISTS friendships;
DROP TABLE IF EXISTS friend_requests;
