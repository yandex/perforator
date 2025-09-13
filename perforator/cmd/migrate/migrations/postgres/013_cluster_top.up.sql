CREATE TABLE IF NOT EXISTS cluster_top_services (
    service TEXT NOT NULL,
    status VARCHAR(100) NOT NULL DEFAULT 'ready',
    profiles_count INT NOT NULL,
    generation INT NOT NULL,
    PRIMARY KEY(service, generation)
);
CREATE INDEX IF NOT EXISTS service_select_index ON cluster_top_services(status, profiles_count);

CREATE TABLE IF NOT EXISTS cluster_top_generations (
    id SERIAL PRIMARY KEY,
    from_ts TIMESTAMPTZ NOT NULL,
    to_ts TIMESTAMPTZ NOT NULL
);
