GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.23.2)

SRCS(
    config.go
)

GO_TEST_SRCS(config_test.go)

END()

RECURSE(
    gotest
)
