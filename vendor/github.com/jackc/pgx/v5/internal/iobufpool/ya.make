GO_LIBRARY()

LICENSE(MIT)

VERSION(v5.7.1)

SRCS(
    iobufpool.go
)

GO_TEST_SRCS(iobufpool_internal_test.go)

GO_XTEST_SRCS(iobufpool_test.go)

END()

RECURSE(
    gotest
)
