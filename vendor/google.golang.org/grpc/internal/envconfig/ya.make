GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.73.0)

SRCS(
    envconfig.go
    observability.go
    xds.go
)

GO_TEST_SRCS(envconfig_test.go)

END()

RECURSE(
    gotest
)
