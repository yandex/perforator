GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.73.0)

SRCS(
    config_selector.go
)

GO_TEST_SRCS(config_selector_test.go)

END()

RECURSE(
    delegatingresolver
    dns
    gotest
    passthrough
    unix
)
