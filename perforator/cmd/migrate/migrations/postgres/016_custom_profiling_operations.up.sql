CREATE TABLE IF NOT EXISTS custom_profiling_operations (
    id TEXT PRIMARY KEY,
    meta JSONB NOT NULL,
    spec JSONB NOT NULL,
    status JSONB,
    target_state JSONB
);
