ALTER TABLE rooms DROP CONSTRAINT IF EXISTS rooms_name_key;

CREATE UNIQUE INDEX IF NOT EXISTS idx_rooms_public_name_unique
    ON rooms (name) WHERE type = 'public';

CREATE INDEX IF NOT EXISTS idx_rooms_type ON rooms (type);
