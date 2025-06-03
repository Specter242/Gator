-- name: CreateUser :one
INSERT INTO users (name)
VALUES ($1)
RETURNING id, created_at, updated_at, name;

-- name: GetUser :one
SELECT * FROM users
WHERE name = $1;

-- name: GetUserById :one
SELECT * FROM users
WHERE id = $1;

-- name: Reset :exec
DELETE FROM users;

-- name: GetUsers :many
SELECT * FROM users
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: CreateFeed :one
INSERT INTO feeds (name, url, user_id)
VALUES ($1, $2, $3)
RETURNING id, created_at, updated_at, name, url, user_id;

-- name: CreateFeedFollow :many
WITH inserted_feed_follows AS (
    INSERT INTO feed_follows (user_id, feed_id)
    VALUES ($1, $2)
    RETURNING *
)
SELECT
    inserted_feed_follows.*,
    feeds.name AS feed_name,
    users.name AS user_name
FROM inserted_feed_follows
JOIN feeds ON inserted_feed_follows.feed_id = feeds.id
JOIN users ON inserted_feed_follows.user_id = users.id;

-- name: GetFeedFollowsForUser :many
SELECT
    feed_follows.*,
    feeds.name AS feed_name,
    users.name AS user_name
FROM feed_follows
JOIN feeds ON feed_follows.feed_id = feeds.id
JOIN users ON feed_follows.user_id = users.id
WHERE feed_follows.user_id = $1
ORDER BY feed_follows.created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetFeeds :many
SELECT * FROM feeds
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: RemoveFeedFollow :exec
DELETE FROM feed_follows
WHERE user_id = $1 AND feed_id = $2;

-- name: LastFetchedAt :exec
UPDATE feeds
SET last_fetched_at = NOW(),
    updated_at = NOW()
WHERE id = $1;

-- name: GetNextFeedToFetch :one
SELECT * FROM feeds
WHERE last_fetched_at IS NULL OR last_fetched_at < NOW() - INTERVAL '1 hour'
ORDER BY last_fetched_at ASC
LIMIT 1;