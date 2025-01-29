GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    doc.go
    interface.go
    parser.go
)

GO_TEST_SRCS(
    error_test.go
    interface_test.go
    parser_test.go
    performance_test.go
    short_test.go
)

GO_XTEST_SRCS(example_test.go)

END()

RECURSE(
    gotest
)
