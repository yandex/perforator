CREATE TABLE IF NOT EXISTS cluster_top_v3
(
    generation UInt32,
    partition_bucket UInt16,
    event_type LowCardinality(String),
    service String CODEC(ZSTD(1)),
    function String CODEC(ZSTD(1)),
    language LowCardinality(String),
    binary_path String CODEC(ZSTD(1)),
    build_id String CODEC(ZSTD(1)),
    commit_id String CODEC(ZSTD(1)),
    source_file String CODEC(ZSTD(1)),
    source_line UInt32,
    source_column UInt32,
    self_cycles UInt128,
    cumulative_cycles UInt128
)
ENGINE = ReplicatedSummingMergeTree(
    '/clickhouse/tables/{shard}/{database}/cluster_top_v3',
    '{replica}',
    (self_cycles, cumulative_cycles)
)
PARTITION BY (generation, event_type, partition_bucket)
PRIMARY KEY (generation, partition_bucket, event_type, function, service)
ORDER BY (
    generation,
    partition_bucket,
    event_type,
    function,
    service,
    language,
    commit_id,
    binary_path,
    build_id,
    source_file,
    source_line,
    source_column
)
SETTINGS index_granularity = 8192;

CREATE TABLE IF NOT EXISTS cluster_top_by_function_v3
(
    generation UInt32,
    partition_bucket UInt16,
    event_type LowCardinality(String),
    function String CODEC(ZSTD(1)),
    self_cycles UInt128,
    cumulative_cycles UInt128
)
ENGINE = ReplicatedSummingMergeTree(
    '/clickhouse/tables/{shard}/{database}/cluster_top_by_function_v3',
    '{replica}',
    (self_cycles, cumulative_cycles)
)
PARTITION BY (generation, event_type, partition_bucket)
PRIMARY KEY (generation, partition_bucket, event_type, function)
ORDER BY (generation, partition_bucket, event_type, function)
SETTINGS index_granularity = 8192;

CREATE MATERIALIZED VIEW IF NOT EXISTS cluster_top_by_function_v3_mv TO cluster_top_by_function_v3 AS
SELECT
    generation,
    partition_bucket,
    event_type,
    function,
    self_cycles,
    cumulative_cycles
FROM cluster_top_v3;
