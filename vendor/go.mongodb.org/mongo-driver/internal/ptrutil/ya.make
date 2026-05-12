GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.17.4)

SRCS(
    int64.go
)

GO_TEST_SRCS(int64_test.go)

END()

RECURSE(
    gotest
)
