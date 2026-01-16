-- name: CreateFeed :one
INSERT INTO feeds (id, created_at, updated_at, name, url, user_id)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6
)
RETURNING *;

-- name: GetAllFeeds :many
SELECT id, created_at, updated_at, name, url, user_id
FROM feeds;

-- name: GetFeedByUrl :one
SELECT id, created_at, updated_at, name, url, user_id
FROM feeds
WHERE Url = $1;

-- name: GetFeedNameByFeedID :one
SELECT name
FROM feeds
WHERE id = $1;

-- name: MarkFeedFetchedByID :exec
UPDATE feeds
SET last_fetched_at = $1, updated_at = $2
WHERE id = $3;

-- name: GetNextFeedToFetchByUserID :one
SELECT id, created_at, updated_at, name, url, user_id
FROM feeds
WHERE user_id = $1
ORDER BY last_fetched_at ASC NULLS FIRST
LIMIT 1;
