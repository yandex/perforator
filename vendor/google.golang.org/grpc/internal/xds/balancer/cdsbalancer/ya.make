GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.80.0)

SRCS(
    cdsbalancer.go
    configbuilder.go
    configbuilder_childname.go
    logging.go
)

GO_TEST_SRCS(
    # aggregate_cluster_test.go
    # cdsbalancer_security_test.go
    # cdsbalancer_test.go
    # configbuilder_childname_test.go
    # configbuilder_test.go
)

END()

RECURSE(
    gotest
)
