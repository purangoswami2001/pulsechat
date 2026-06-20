-- name: CreateUser :one
INSERT INTO users (id, username, email, password_hash, created_at)
VALUES ($1, $2, $3, $4, COALESCE(sqlc.narg('created_at'), CURRENT_TIMESTAMP))
RETURNING id, username, email, created_at;

-- name: GetUserByEmail :one
SELECT id, username, email, password_hash, created_at
FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT id, username, email, created_at
FROM users
WHERE id = $1;
