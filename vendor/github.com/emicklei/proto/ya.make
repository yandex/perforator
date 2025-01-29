GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.11.0)

SRCS(
    comment.go
    enum.go
    extensions.go
    field.go
    group.go
    import.go
    message.go
    noop_visitor.go
    oneof.go
    option.go
    package.go
    parent_accessor.go
    parser.go
    proto.go
    range.go
    reserved.go
    service.go
    syntax.go
    token.go
    visitor.go
    walk.go
)

GO_TEST_SRCS(
    comment_test.go
    enum_test.go
    extensions_test.go
    field_test.go
    group_test.go
    import_test.go
    message_test.go
    oneof_test.go
    option_test.go
    package_test.go
    parent_test.go
    parser_test.go
    protobuf_test.go
    range_test.go
    reserved_test.go
    service_test.go
    syntax_test.go
    token_test.go
    visitor_test.go
    walk_test.go
)

END()

RECURSE(
    gotest
)
