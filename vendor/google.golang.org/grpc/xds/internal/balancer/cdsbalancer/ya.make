GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.73.0)

SRCS(
    cdsbalancer.go
    cluster_watcher.go
    logging.go
)

GO_TEST_SRCS(
    aggregate_cluster_test.go
    cdsbalancer_security_test.go
    cdsbalancer_test.go
)

END()

RECURSE(
    gotest
)
