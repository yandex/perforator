GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.79.1)

SRCS(
    config.go
    gracefulswitch.go
)

GO_TEST_SRCS(gracefulswitch_test.go)

END()

RECURSE(
    gotest
)
