GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v2.14.1)

SRCS(
    grpclog.go
)

GO_TEST_SRCS(grpclog_test.go)

END()

RECURSE(
    gotest
)
