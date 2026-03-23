DROP INDEX IF EXISTS gsym_last_used_idx;
ALTER TABLE gsym DROP COLUMN IF EXISTS last_used_timestamp;
