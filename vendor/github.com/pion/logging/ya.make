GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.2.3)

SRCS(
    logger.go
    scoped.go
)

GO_XTEST_SRCS(logging_test.go)

END()

RECURSE(
    gotest
)
