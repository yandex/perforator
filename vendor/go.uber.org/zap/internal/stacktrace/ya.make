GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.28.0)

SRCS(
    stack.go
)

GO_TEST_SRCS(stack_test.go)

END()

RECURSE(
    gotest
)
