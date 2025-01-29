GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.1.0)

SRCS(
    context.go
)

GO_TEST_SRCS(context_test.go)

END()

RECURSE(
    gotest
    logtest
)
