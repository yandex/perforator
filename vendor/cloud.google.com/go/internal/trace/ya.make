GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.112.2)

SRCS(
    trace.go
)

GO_TEST_SRCS(trace_test.go)

END()

RECURSE(
    gotest
)
