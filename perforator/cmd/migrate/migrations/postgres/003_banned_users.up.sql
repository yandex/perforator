CREATE TABLE IF NOT EXISTS banned_users (
    login text PRIMARY KEY,
    banned_since timestamptz DEFAULT NOW() NOT NULL,
    banned_due timestamptz
);
