GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.34.3)

SRCS(
    collections.go
    doc.go
    protobuf.go
)

GO_TEST_SRCS(
    collections_test.go
    protobuf_test.go
)

END()

RECURSE(
    gotest
)
