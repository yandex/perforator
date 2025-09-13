GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v2.26.3)

SRCS(
    context.go
    convert.go
    doc.go
    errors.go
    fieldmask.go
    handler.go
    marshal_httpbodyproto.go
    marshal_json.go
    marshal_jsonpb.go
    marshal_proto.go
    marshaler.go
    marshaler_registry.go
    mux.go
    pattern.go
    proto2_convert.go
    query.go
)

GO_TEST_SRCS(
    # fieldmask_test.go
    # mux_internal_test.go
    # pattern_test.go
)

GO_XTEST_SRCS(
    # context_test.go
    # convert_test.go
    # errors_test.go
    # handler_test.go
    # marshal_httpbodyproto_test.go
    # marshal_json_test.go
    # marshal_jsonpb_test.go
    # marshal_proto_test.go
    # marshaler_registry_test.go
    # mux_test.go
    # query_fuzz_test.go
    # query_test.go
)

END()

RECURSE(
    gotest
    internal
)
