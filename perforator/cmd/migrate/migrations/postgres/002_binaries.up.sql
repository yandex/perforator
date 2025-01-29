CREATE TABLE IF NOT EXISTS binaries(
    build_id text PRIMARY KEY,
    blob_size bigint NOT NULL,
    ts timestamptz NOT NULL,
    attributes jsonb,
    upload_status text NOT NULL,
    last_used_timestamp timestamptz DEFAULT NOW() NOT NULL
);
