GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.7.0)

SRCS(
    streams.go
)

GO_TEST_SRCS(streams_test.go)

END()

RECURSE(
    gotest
)
