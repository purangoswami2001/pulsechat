DROP INDEX IF EXISTS idx_rooms_type;
DROP INDEX IF EXISTS idx_rooms_public_name_unique;

ALTER TABLE rooms ADD CONSTRAINT rooms_name_key UNIQUE (name);
