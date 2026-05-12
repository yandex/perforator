GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.33.0)

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
