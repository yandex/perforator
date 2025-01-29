GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.17.1)

SRCS(
    randutil.go
)

GO_TEST_SRCS(randutil_test.go)

END()

RECURSE(
    gotest
)
