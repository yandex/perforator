GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.26.1)

SRCS(
    from_stack.go
)

GO_TEST_SRCS(from_stack_test.go)

END()

RECURSE(
    gotest
)
