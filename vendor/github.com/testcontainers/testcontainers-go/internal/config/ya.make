GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.32.0)

SRCS(
    config.go
)

GO_TEST_SRCS(config_test.go)

END()

RECURSE(
    gotest
)
