CREATE TABLE IF NOT EXISTS gsym (
    build_id TEXT NOT NULL,
    uncompressed_size BIGINT NOT NULL DEFAULT 0,
    compressed_size BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY(build_id)
);
