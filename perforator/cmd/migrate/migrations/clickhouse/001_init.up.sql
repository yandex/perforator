CREATE TABLE profiles
(
    id UUID CODEC(ZSTD(1)),
    system_name LowCardinality(String),
    event_type LowCardinality(String),
    cluster LowCardinality(String),
    service String CODEC(ZSTD(1)),
    pod_id String CODEC(ZSTD(1)),
    node_id String CODEC(ZSTD(1)),
    timestamp DateTime64(3) CODEC(DoubleDelta, ZSTD(1)),
    build_ids Array(String) CODEC(ZSTD(3)),
    attributes Map(LowCardinality(String), String) CODEC(ZSTD(3)),

    INDEX node_index node_id TYPE set(1024) GRANULARITY 1,
    INDEX build_id_index build_ids TYPE set(1024) GRANULARITY 1,
    INDEX attributes_key_index mapKeys(attributes) TYPE ngrambf_v1(3, 256, 3, 0) GRANULARITY 1,
    INDEX attributes_value_index mapValues(attributes) TYPE ngrambf_v1(3, 256, 3, 0) GRANULARITY 1,
    INDEX timestamp_index (service, timestamp) TYPE minmax GRANULARITY 1,

    PROJECTION service_names
    (
        SELECT
            service AS service,
            max(timestamp) AS max_timestamp,
            sum(1) AS profile_count
        GROUP BY service
    )
)
ENGINE = ReplicatedMergeTree(
    '/clickhouse/tables/{shard}/{database}/{table}',
    '{replica}'
)
PARTITION BY (system_name, event_type, toStartOfDay(timestamp))
PRIMARY KEY (system_name, event_type, cluster, service, pod_id)
ORDER BY (system_name, event_type, cluster, service, pod_id, timestamp)
SETTINGS index_granularity = 8192
