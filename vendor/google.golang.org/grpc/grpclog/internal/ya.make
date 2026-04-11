GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.79.1)

SRCS(
    grpclog.go
    logger.go
    loggerv2.go
)

GO_TEST_SRCS(loggerv2_test.go)

END()

RECURSE(
    gotest
)
