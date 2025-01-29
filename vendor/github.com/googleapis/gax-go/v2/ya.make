GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v2.12.3)

SRCS(
    call_option.go
    content_type.go
    gax.go
    header.go
    invoke.go
    proto_json_stream.go
)

GO_TEST_SRCS(
    call_option_test.go
    content_type_test.go
    header_test.go
    invoke_test.go
    proto_json_stream_test.go
)

GO_XTEST_SRCS(example_test.go)

END()

RECURSE(
    apierror
    callctx
    gotest
    internal
)
