CREATE TABLE IF NOT EXISTS microscopes(
    id uuid PRIMARY KEY,
    user_id varchar(100) NOT NULL,
    selector text NOT NULL,
    from_ts timestamptz NOT NULL,
    to_ts timestamptz NOT NULL,
    created_at timestamptz DEFAULT NOW() NOT NULL
);
CREATE INDEX IF NOT EXISTS microscope_from_ts_index ON microscopes(from_ts, created_at);
CREATE INDEX IF NOT EXISTS microscope_user_index ON microscopes(user_id, created_at);
