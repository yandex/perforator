CREATE TABLE cluster_top_by_function_v2
(
    generation UInt32,
    function String CODEC(ZSTD(1)),
    self_cycles UInt128,
    cumulative_cycles UInt128
)
ENGINE = ReplicatedSummingMergeTree(
    '/clickhouse/tables/{shard}/{database}/cluster_top_by_function_v2',
    '{replica}',
    (self_cycles, cumulative_cycles)
)
PARTITION BY generation
ORDER BY (generation, function)
SETTINGS index_granularity = 8192;

CREATE MATERIALIZED VIEW cluster_top_by_function_v2_mv TO cluster_top_by_function_v2 AS
SELECT
    generation,
    function,
    self_cycles,
    cumulative_cycles
FROM cluster_top_v2;
