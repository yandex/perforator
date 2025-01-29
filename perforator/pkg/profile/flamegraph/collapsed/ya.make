GO_LIBRARY()

SRCS(
    stacks.go
)

GO_XTEST_SRCS(stacks_test.go)

END()

RECURSE(
    gotest
)
