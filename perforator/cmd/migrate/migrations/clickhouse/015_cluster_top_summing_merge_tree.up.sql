CREATE TABLE cluster_top_v2
(
    generation UInt32,
    service String CODEC(ZSTD(1)),
    function String CODEC(ZSTD(1)),
    self_cycles UInt128,
    cumulative_cycles UInt128,

    PROJECTION proj_by_function_service
    (
        SELECT
            generation,
            function,
            service,
            self_cycles,
            cumulative_cycles
        ORDER BY (generation, function, service)
    )
)
ENGINE = ReplicatedSummingMergeTree(
    '/clickhouse/tables/{shard}/{database}/cluster_top_v2',
    '{replica}',
    (self_cycles, cumulative_cycles)
)
PARTITION BY generation
ORDER BY (generation, service, function)
SETTINGS index_granularity = 8192, deduplicate_merge_projection_mode = 'drop';
