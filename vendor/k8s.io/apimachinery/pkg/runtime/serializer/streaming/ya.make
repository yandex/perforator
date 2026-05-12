GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.33.0)

SRCS(
    streaming.go
)

GO_TEST_SRCS(streaming_test.go)

END()

RECURSE(
    gotest
)
