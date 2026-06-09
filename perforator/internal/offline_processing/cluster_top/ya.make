GO_LIBRARY()

SRCS(
    clickhouse_perf_top_aggregator.go
    config.go
    cluster_top.go
    execution_stats.go
    metrics.go
    models.go
    one_shot_job_processor.go
    pg_job_selector.go
)

IF (NOT OPENSOURCE)
    GO_TEST_SRCS(
        pg_job_selector_test.go
    )
ENDIF()

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

IF (NOT OPENSOURCE)
    RECURSE(
        gotest
    )
ENDIF()

RECURSE(
    scheduler
)
