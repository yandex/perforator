GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.0)

SRCS(
    api.go
    astnodetype_string.go
    functions.go
    interpreter.go
    lexer.go
    parser.go
    toktype_string.go
    util.go
)

GO_TEST_SRCS(
    api_test.go
    compliance_test.go
    interpreter_test.go
    lexer_test.go
    parser_test.go
    util_test.go
)

END()

RECURSE(
    cmd
    fuzz
    gotest
)
