GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    format.go
    import.go
    node.go
    printer.go
    simplify.go
)

GO_TEST_SRCS(
    format_test.go
    node_test.go
)

END()

RECURSE(
    gotest
)
