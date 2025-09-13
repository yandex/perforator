GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.27.0)

SRCS(
    catmsg.go
    codec.go
    varint.go
)

GO_TEST_SRCS(
    catmsg_test.go
    varint_test.go
)

END()

RECURSE(
    gotest
)
