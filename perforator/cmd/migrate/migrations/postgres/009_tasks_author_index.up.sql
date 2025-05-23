CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_tasks_by_author ON tasks USING btree ((meta->>'Author'));
