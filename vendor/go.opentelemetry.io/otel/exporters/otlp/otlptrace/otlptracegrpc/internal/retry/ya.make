GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.35.0)

SRCS(
    retry.go
)

GO_TEST_SRCS(retry_test.go)

END()

RECURSE(
    gotest
)
