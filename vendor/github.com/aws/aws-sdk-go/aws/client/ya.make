GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.46.7)

SRCS(
    client.go
    default_retryer.go
    logger.go
    no_op_retryer.go
)

GO_TEST_SRCS(
    client_test.go
    default_retryer_test.go
    logger_test.go
    no_op_retryer_test.go
)

END()

RECURSE(
    gotest
    metadata
)
