GO_LIBRARY()

LICENSE(MIT)

VERSION(v2.2.2)

SRCS(
    gen_stack.go
    stack.go
)

GO_TEST_SRCS(gen_stack_test.go)

END()

RECURSE(
    gotest
)
