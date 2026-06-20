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
