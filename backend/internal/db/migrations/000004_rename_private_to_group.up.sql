-- Rename private rooms to group; app no longer uses public channels
UPDATE rooms SET type = 'group' WHERE type = 'private';
