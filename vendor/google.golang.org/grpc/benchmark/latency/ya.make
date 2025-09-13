GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.73.0)

SRCS(
    latency.go
)

GO_TEST_SRCS(latency_test.go)

END()

RECURSE(
    gotest
)
