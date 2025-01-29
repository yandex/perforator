ALTER TABLE profiles
ADD COLUMN IF NOT EXISTS expired Boolean DEFAULT (false) TTL toDateTime(timestamp) + INTERVAL 14 DAY
