GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.5.1)

SRCS(
    assertion_format.go
    assertion_forward.go
    assertion_order.go
    assertions.go
    doc.go
    errors.go
    forward_assertions.go
    http_assertions.go
)

GO_TEST_SRCS(
    assertion_order_test.go
    assertions_test.go
    forward_assertions_test.go
    http_assertions_test.go
)

END()

RECURSE(
    #gotest
)
