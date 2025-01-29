GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.27.0)

SRCS(
    stack.go
)

GO_TEST_SRCS(stack_test.go)

END()

RECURSE(
    gotest
)
