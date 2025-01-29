GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v1.3.2)

SRCS(
    any.go
    any.pb.go
    api.pb.go
    doc.go
    duration.go
    duration.pb.go
    duration_gogo.go
    empty.pb.go
    field_mask.pb.go
    protosize.go
    source_context.pb.go
    struct.pb.go
    timestamp.go
    timestamp.pb.go
    timestamp_gogo.go
    type.pb.go
    wrappers.pb.go
    wrappers_gogo.go
)

GO_TEST_SRCS(
    any_test.go
    duration_test.go
    timestamp_test.go
)

END()

RECURSE(
    gotest
)
