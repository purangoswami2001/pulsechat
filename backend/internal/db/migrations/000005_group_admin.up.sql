ALTER TABLE room_members ADD COLUMN IF NOT EXISTS is_admin BOOLEAN NOT NULL DEFAULT false;

-- Group creators (first member) become admin
UPDATE room_members rm
SET is_admin = true
FROM rooms r
WHERE rm.room_id = r.id
  AND r.type IN ('group', 'private')
  AND rm.user_id = (
    SELECT rm2.user_id
    FROM room_members rm2
    WHERE rm2.room_id = r.id
    ORDER BY rm2.joined_at ASC
    LIMIT 1
  );
