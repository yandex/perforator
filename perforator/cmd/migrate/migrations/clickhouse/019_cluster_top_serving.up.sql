CREATE TABLE IF NOT EXISTS cluster_top_by_function_v3_serving
(
    generation UInt32,
    event_type LowCardinality(String),
    -- One copy per ranking keeps both orders in a single generation partition,
    -- so REPLACE PARTITION publishes the complete snapshot in one operation.
    sort_type LowCardinality(String),
    sort_cycles UInt128,
    function String CODEC(ZSTD(1)),
    self_cycles UInt128,
    cumulative_cycles UInt128,
    -- Bloom preserves substring LIKE filtering with a bounded index size.
    -- Text indexes used substantially more disk in our serving-table benchmark.
    INDEX function_search function TYPE ngrambf_v1(3, 16384, 4, 0) GRANULARITY 1
)
ENGINE = ReplicatedMergeTree(
    '/clickhouse/tables/{shard}/{database}/cluster_top_by_function_v3_serving',
    '{replica}'
)
PARTITION BY generation
ORDER BY (generation, event_type, sort_type, sort_cycles, function)
SETTINGS index_granularity = 1024;

