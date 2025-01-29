CREATE TABLE IF NOT EXISTS tasks (
    id UUID PRIMARY KEY,
    idempotency_key TEXT,
    meta JSONB NOT NULL,
    spec JSONB NOT NULL,
    status JSONB NOT NULL,
    result JSONB NOT NULL
);

CREATE INDEX idx_idempotency_key ON tasks(idempotency_key);