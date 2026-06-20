-- name: CreateUser :exec
INSERT INTO users (id, username, email, password_hash, created_at)
VALUES (?, ?, ?, ?, COALESCE(?, CURRENT_TIMESTAMP));

-- name: GetUserByEmail :one
SELECT id, username, email, password_hash, created_at
FROM users
WHERE email = ?;

-- name: GetUserByID :one
SELECT id, username, email, created_at
FROM users
WHERE id = ?;
