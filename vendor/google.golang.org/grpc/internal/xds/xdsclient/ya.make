GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.79.1)

SRCS(
    attributes.go
    client.go
    clientimpl.go
    clientimpl_loadreport.go
    logging.go
    pool.go
    requests_counter.go
    resource_types.go
)

GO_TEST_SRCS(
    # client_refcounted_test.go
    # client_test.go
    # clientimpl_test.go
    # metrics_test.go
    # requests_counter_test.go
)

GO_XTEST_SRCS(
    # xdsclient_test.go
)

END()

RECURSE(
    gotest
    internal
    pool
    xdslbregistry
    xdsresource
)
