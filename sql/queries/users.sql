-- name: CreateUser :one
INSERT INTO users (name)
VALUES ($1)
RETURNING id, created_at, updated_at, name;

-- name: GetUser :one
SELECT * FROM users
WHERE name = $1;

-- name: Reset :exec
DELETE FROM users;

-- name: GetUsers :many
SELECT * FROM users
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;