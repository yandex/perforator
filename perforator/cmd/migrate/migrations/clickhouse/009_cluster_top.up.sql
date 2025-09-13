CREATE TABLE cluster_top
(
    generation UInt32,
    service String CODEC(ZSTD(1)),
    function String CODEC(ZSTD(1)),
    self_cycles UInt128,
    cumulative_cycles UInt128
)
ENGINE = ReplicatedMergeTree(
    '/clickhouse/tables/{shard}/{database}/{table}',
    '{replica}'
)
PARTITION BY (generation)
PRIMARY KEY (generation, service)
