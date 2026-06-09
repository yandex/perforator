ALTER TABLE cluster_top_jobs
    DROP COLUMN IF EXISTS execution_stats,
    DROP COLUMN IF EXISTS finished_at,
    DROP COLUMN IF EXISTS started_at,
    DROP COLUMN IF EXISTS created_at;
