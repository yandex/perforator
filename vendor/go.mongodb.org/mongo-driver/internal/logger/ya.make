GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.17.1)

SRCS(
    component.go
    context.go
    io_sink.go
    level.go
    logger.go
)

GO_TEST_SRCS(
    component_test.go
    logger_test.go
)

GO_XTEST_SRCS(context_test.go)

END()

RECURSE(
    gotest
)
