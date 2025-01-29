GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    errors.go
    parse.go
    protobuf.go
    types.go
    util.go
)

GO_TEST_SRCS(protobuf_test.go)

GO_XTEST_SRCS(examples_test.go)

END()

RECURSE(
    gotest
    jsonpb
    pbinternal
    textproto
)
