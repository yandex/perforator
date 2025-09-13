GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.35.0)

SRCS(
    envconfig.go
)

GO_TEST_SRCS(envconfig_test.go)

END()

RECURSE(
    gotest
)
