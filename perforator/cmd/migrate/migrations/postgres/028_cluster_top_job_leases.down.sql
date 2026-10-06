-- Stop lease-based workers before rolling back this migration.
UPDATE cluster_top_jobs
SET status = 'pending', started_at = NULL
WHERE status = 'running';

ALTER TABLE cluster_top_jobs
    DROP COLUMN lease_token,
    DROP COLUMN lease_expires_at;
