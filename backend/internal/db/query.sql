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

-- name: CreateRoom :one
INSERT INTO rooms (id, name, type, created_at)
VALUES ($1, $2, $3, COALESCE(sqlc.narg('created_at'), CURRENT_TIMESTAMP))
RETURNING id, name, type, created_at;

-- name: GetRoomByID :one
SELECT id, name, type, created_at
FROM rooms
WHERE id = $1;

-- name: ListRooms :many
SELECT id, name, type, created_at
FROM rooms
ORDER BY created_at DESC;

-- name: CreateRoomMember :exec
INSERT INTO room_members (room_id, user_id, joined_at)
VALUES ($1, $2, COALESCE(sqlc.narg('joined_at'), CURRENT_TIMESTAMP));

-- name: GetRoomMembers :many
SELECT u.id, u.username, u.email, rm.joined_at
FROM room_members rm
JOIN users u ON rm.user_id = u.id
WHERE rm.room_id = $1
ORDER BY u.username ASC;

-- name: CreateMessage :one
INSERT INTO messages (id, room_id, sender_id, content, created_at)
VALUES ($1, $2, $3, $4, COALESCE(sqlc.narg('created_at'), CURRENT_TIMESTAMP))
RETURNING id, room_id, sender_id, content, created_at;

-- name: ListMessagesByRoom :many
SELECT m.id, m.room_id, m.sender_id, u.username AS sender_name, u.avatar_url AS sender_avatar_url, m.content, m.attachment_url, m.attachment_type, m.created_at
FROM messages m
JOIN users u ON m.sender_id = u.id
WHERE m.room_id = $1
ORDER BY m.created_at ASC
LIMIT $2 OFFSET $3;
