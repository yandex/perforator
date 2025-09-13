GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.7.8)

SRCS(
    markdown.go
)

GO_XTEST_SRCS(
    ast_test.go
    commonmark_test.go
    extra_test.go
    options_test.go
)

END()

RECURSE(
    ast
    extension
    fuzz
    gotest
    parser
    renderer
    testutil
    text
    util
)
