CREATE TABLE cluster_top_generation_totals_v2
(
    generation UInt32,
    total_self_cycles UInt128
)
ENGINE = ReplicatedSummingMergeTree(
    '/clickhouse/tables/{shard}/{database}/cluster_top_generation_totals_v2',
    '{replica}',
    total_self_cycles
)
ORDER BY generation
SETTINGS index_granularity = 8192;

CREATE MATERIALIZED VIEW cluster_top_generation_totals_v2_mv TO cluster_top_generation_totals_v2 AS
SELECT
    generation,
    self_cycles AS total_self_cycles
FROM cluster_top_v2;
