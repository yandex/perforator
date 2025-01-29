CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_pending_tasks ON tasks ((status->>'State')) WHERE (status->>'State' = 'Running' OR status->>'State' = 'Created')
