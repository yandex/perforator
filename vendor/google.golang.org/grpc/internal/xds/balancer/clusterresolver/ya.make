GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.79.3)

SRCS(
    clusterresolver.go
    config.go
    configbuilder.go
    configbuilder_childname.go
    logging.go
    resource_resolver.go
    resource_resolver_dns.go
    resource_resolver_eds.go
)

GO_TEST_SRCS(
    # clusterresolver_test.go
    # config_test.go
    # configbuilder_childname_test.go
    # configbuilder_test.go
)

END()

RECURSE(
    gotest
)
