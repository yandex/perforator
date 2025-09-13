CREATE TABLE custom_profiles
(
    id UUID CODEC(ZSTD(1)),
    operation_id UUID CODEC(ZSTD(1)),
    from_timestamp DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    to_timestamp DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    build_ids Array(String) CODEC(ZSTD(3)),
    attributes Map(LowCardinality(String), String) CODEC(ZSTD(3)),
)
ENGINE = ReplicatedMergeTree(
    '/clickhouse/tables/{shard}/{database}/{table}',
    '{replica}'
)
PARTITION BY (toStartOfMonth(from_timestamp), sipHash64(operation_id) % 64)
PRIMARY KEY (operation_id, from_timestamp)
ORDER BY (operation_id, from_timestamp, id)
TTL toDateTime(from_timestamp) + INTERVAL 3 MONTH
