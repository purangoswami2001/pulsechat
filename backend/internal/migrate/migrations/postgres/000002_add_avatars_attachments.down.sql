ALTER TABLE users DROP COLUMN IF EXISTS avatar_url;
ALTER TABLE messages DROP COLUMN IF EXISTS attachment_url;
ALTER TABLE messages DROP COLUMN IF EXISTS attachment_type;
