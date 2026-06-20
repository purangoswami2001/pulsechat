-- name: CreateRoom :exec
INSERT INTO rooms (id, name, type, created_at)
VALUES (?, ?, ?, COALESCE(?, CURRENT_TIMESTAMP));

-- name: GetRoomByID :one
SELECT id, name, type, created_at
FROM rooms
WHERE id = ?;

-- name: ListRooms :many
SELECT id, name, type, created_at
FROM rooms
ORDER BY created_at DESC;

-- name: CreateRoomMember :exec
INSERT INTO room_members (room_id, user_id, joined_at)
VALUES (?, ?, COALESCE(?, CURRENT_TIMESTAMP));

-- name: GetRoomMembers :many
SELECT u.id, u.username, u.email, rm.joined_at
FROM room_members rm
JOIN users u ON rm.user_id = u.id
WHERE rm.room_id = ?
ORDER BY u.username ASC;
