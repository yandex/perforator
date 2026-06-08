CREATE TABLE IF NOT EXISTS cluster_top_jobs (
    id BIGSERIAL PRIMARY KEY,
    generation INT NOT NULL,
    service TEXT NOT NULL,
    pod_id TEXT NOT NULL DEFAULT '',
    node_id TEXT NOT NULL DEFAULT '',
    profiles_count INT NOT NULL,
    status VARCHAR(100) NOT NULL DEFAULT 'pending'
);

CREATE UNIQUE INDEX IF NOT EXISTS cluster_top_jobs_generation_service_workload_idx
    ON cluster_top_jobs (generation, service, (coalesce(nullif(pod_id, ''), node_id)));

CREATE INDEX IF NOT EXISTS cluster_top_jobs_pending_profiles_count_idx
    ON cluster_top_jobs (profiles_count DESC)
    WHERE status = 'pending';
