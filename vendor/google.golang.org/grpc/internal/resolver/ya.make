GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.63.2)

SRCS(
    config_selector.go
)

GO_TEST_SRCS(config_selector_test.go)

END()

RECURSE(
    dns
    gotest
    passthrough
    unix
)
