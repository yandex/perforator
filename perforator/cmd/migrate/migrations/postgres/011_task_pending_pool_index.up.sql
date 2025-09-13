CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_pending_tasks_by_pool ON tasks((meta->>'Pool'), (status->>'State')) WHERE (status->>'State' = 'Running' OR status->>'State' = 'Created');
