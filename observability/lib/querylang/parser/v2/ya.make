GO_LIBRARY()

SRCS(
    error_listener.go
    expression_listener.go
    operators.go
    parse_error.go
    parser.go
    selector_listener.go
    utils.go
)

GO_XTEST_SRCS(
    parse_error_test.go
    parser_test.go
    utils_test.go
)

END()

RECURSE(
    generated
    gotest
)
