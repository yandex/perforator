CREATE INDEX CONCURRENTLY cluster_top_jobs_pending_profiles_count_idx
    ON cluster_top_jobs (profiles_count DESC)
    WHERE status = 'pending';
