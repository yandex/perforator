GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.6.0)

SRCS(
    tracing.go
)

GO_TEST_SRCS(tracing_test.go)

END()

RECURSE(
    gotest
)
