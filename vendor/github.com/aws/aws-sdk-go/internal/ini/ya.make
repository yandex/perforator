GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.46.7)

SRCS(
    ast.go
    comma_token.go
    comment_token.go
    doc.go
    empty_token.go
    expression.go
    ini.go
    ini_lexer.go
    ini_parser.go
    literal_tokens.go
    newline_token.go
    number_helper.go
    op_tokens.go
    parse_error.go
    parse_stack.go
    sep_tokens.go
    skipper.go
    statement.go
    value_util.go
    visitor.go
    walker.go
    ws_token.go
)

GO_TEST_SRCS(
    bench_test.go
    ini_lexer_test.go
    ini_parser_test.go
    literal_tokens_test.go
    number_helper_test.go
    op_tokens_test.go
    parse_stack_test.go
    sep_tokens_test.go
    skipper_test.go
    trim_spaces_test.go
    value_util_test.go
    walker_test.go
)

END()

RECURSE(
    gotest
)
