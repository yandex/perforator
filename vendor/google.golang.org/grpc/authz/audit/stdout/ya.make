GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.79.1)

SRCS(
    stdout_logger.go
)

GO_TEST_SRCS(stdout_logger_test.go)

END()

RECURSE(
    gotest
)
