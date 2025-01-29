GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.63.2)

SRCS(
    bufconn.go
)

GO_TEST_SRCS(bufconn_test.go)

END()

RECURSE(
    gotest
)
