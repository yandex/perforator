GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.34.3)

SRCS(
    streaming.go
)

GO_TEST_SRCS(streaming_test.go)

END()

RECURSE(
    gotest
)
