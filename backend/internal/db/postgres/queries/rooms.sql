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
