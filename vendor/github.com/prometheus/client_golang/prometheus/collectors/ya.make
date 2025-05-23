GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.21.1)

SRCS(
    collectors.go
    dbstats_collector.go
    expvar_collector.go
    go_collector_latest.go
    process_collector.go
)

GO_TEST_SRCS(
    # dbstats_collector_test.go
    go_collector_go123_test.go
    go_collector_latest_test.go
)

END()

RECURSE(
    gotest
    version
)
