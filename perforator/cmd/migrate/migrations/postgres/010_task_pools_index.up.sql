CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_pool ON tasks((meta->>'Pool'));
