GO_LIBRARY()

SRCS(
    clickhouse_perf_top_aggregator.go
    config.go
    cluster_top.go
    models.go
    pg_service_selector.go
)

IF (CGO_ENABLED)
    USE_CXX()

    PEERDIR(
        perforator/symbolizer/lib/symbolize
        perforator/symbolizer/lib/cluster_top
    )

    CGO_SRCS(symbolize.go)
ELSE()
    SRCS(symbolize_stub.go)
ENDIF()

END()
