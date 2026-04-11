GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.14.0)

SRCS(
    default.go
    log.go
    test.go
)

GO_TEST_SRCS(log_test.go)

END()

RECURSE(
    gotest
)
