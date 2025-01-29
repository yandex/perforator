GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    doc.go
    indent.go
    num.go
    quote.go
    string.go
)

GO_TEST_SRCS(
    indent_test.go
    num_test.go
    quote_test.go
    string_test.go
)

END()

RECURSE(
    gotest
)
