GO_LIBRARY()

SRCS(
    listener.go
    operators.go
    parser.go
    utils.go
)

GO_XTEST_SRCS(
    parser_test.go
    utils_test.go
)

END()

RECURSE(
    generated
    gotest
)
