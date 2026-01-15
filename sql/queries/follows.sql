-- name: CreateFeedFollow :one
WITH inserted_feed_follow AS (
    INSERT INTO feed_follows (user_id, feed_id)
    VALUES (
        $1,
        $2
    )
RETURNING *
)
SELECT inserted_feed_follow.*,
    feeds.name AS feed_name,
    users.name AS user_name
FROM inserted_feed_follow
INNER JOIN feeds ON inserted_feed_follow.feed_id = feeds.id
INNER JOIN users ON inserted_feed_follow.user_id = users.id;

-- name: GetFeedFollowForUser :many
SELECT id, created_at, updated_at, user_id, feed_id
FROM feed_follows
WHERE user_id = $1;

-- name: UnfollowFeedByID :exec
DELETE FROM feed_follows
WHERE user_id = $1 AND feed_id = $2;
