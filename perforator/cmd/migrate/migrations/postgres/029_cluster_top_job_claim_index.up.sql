CREATE INDEX CONCURRENTLY cluster_top_jobs_claim_idx
    ON cluster_top_jobs (profiles_count DESC, id)
    WHERE status IN ('pending', 'running');
