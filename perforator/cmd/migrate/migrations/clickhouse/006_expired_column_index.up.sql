ALTER TABLE profiles ADD INDEX IF NOT EXISTS expired_index expired TYPE minmax GRANULARITY 1
