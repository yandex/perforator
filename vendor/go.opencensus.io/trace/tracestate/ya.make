GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.24.0)

SRCS(
    tracestate.go
)

GO_TEST_SRCS(tracestate_test.go)

END()

RECURSE(
    gotest
)
