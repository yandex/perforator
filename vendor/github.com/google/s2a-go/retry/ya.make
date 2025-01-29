GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.1.7)

SRCS(
    retry.go
)

GO_TEST_SRCS(retry_test.go)

END()

RECURSE(
    gotest
)
