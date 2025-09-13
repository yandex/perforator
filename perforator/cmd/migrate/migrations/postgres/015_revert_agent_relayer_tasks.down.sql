CREATE TABLE IF NOT EXISTS agent_relayer_tasks (
    id UUID PRIMARY KEY,
    idempotency_key TEXT,
    meta JSONB NOT NULL,
    spec JSONB NOT NULL,
    status JSONB NOT NULL,
    result JSONB NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_agent_relayer_tasks_idempotency_key ON agent_relayer_tasks(idempotency_key);
CREATE INDEX IF NOT EXISTS idx_agent_relayer_tasks_by_author ON agent_relayer_tasks USING btree ((meta->>'Author'));
