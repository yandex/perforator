GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.2.1)

SRCS(
    logger.go
)

GO_TEST_SRCS(logger_test.go)

END()

RECURSE(
    gotest
)
