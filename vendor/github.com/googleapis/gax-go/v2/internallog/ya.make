GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v2.14.1)

SRCS(
    internallog.go
)

GO_TEST_SRCS(internallog_test.go)

END()

RECURSE(
    gotest
    grpclog
    internal
)
