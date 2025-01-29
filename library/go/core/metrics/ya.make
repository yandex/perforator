GO_LIBRARY()

SRCS(
    buckets.go
    metrics.go
)

GO_TEST_SRCS(buckets_test.go)

END()

RECURSE(
    collect
    gotest
    internal
    mock
    nop
    prometheus
)

IF (NOT OPENSOURCE)
    RECURSE(solomon)
ENDIF()
