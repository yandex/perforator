GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.73.0)

SRCS(
    logging.go
    serviceconfig.go
    watch_service.go
    xds_resolver.go
)

GO_TEST_SRCS(
    # serviceconfig_test.go
)

GO_XTEST_SRCS(
    cluster_specifier_plugin_test.go
    helpers_test.go
    watch_service_test.go
    xds_resolver_test.go
)

END()

RECURSE(
    gotest
    internal
)
