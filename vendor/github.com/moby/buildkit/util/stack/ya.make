GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.20.2)

SRCS(
    compress.go
    stack.go
    stack.pb.go
    stack_vtproto.pb.go
)

GO_TEST_SRCS(compress_test.go)

END()

RECURSE(
    gotest
)
