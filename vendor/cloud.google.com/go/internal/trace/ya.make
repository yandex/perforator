GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.116.0)

SRCS(
    trace.go
)

GO_TEST_SRCS(trace_test.go)

END()

RECURSE(
    gotest
)
