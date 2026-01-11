-- +goose Up
CREATE TABLE feeds (
    id UUID PRIMARY KEY NOT NULL DEFAULT gen_random_uuid(),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    name TEXT,
    url TEXT NOT NULL UNIQUE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE feeds;