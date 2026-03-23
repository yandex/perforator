ALTER TABLE gsym ADD COLUMN last_used_timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW();
CREATE INDEX IF NOT EXISTS gsym_last_used_idx ON gsym(last_used_timestamp);
