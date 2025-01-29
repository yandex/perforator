GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.63.2)

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
