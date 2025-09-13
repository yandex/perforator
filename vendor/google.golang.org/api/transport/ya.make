GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.228.0)

SRCS(
    dial.go
    doc.go
)

GO_XTEST_SRCS(examples_test.go)

END()

RECURSE(
    gotest
    grpc
    http
)
