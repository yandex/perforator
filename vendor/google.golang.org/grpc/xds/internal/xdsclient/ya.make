GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.73.0)

SRCS(
    attributes.go
    authority.go
    channel.go
    client.go
    clientimpl.go
    clientimpl_loadreport.go
    clientimpl_watchers.go
    logging.go
    pool.go
    requests_counter.go
)

GO_TEST_SRCS(
    channel_test.go
    client_refcounted_test.go
    # client_test.go
    metrics_test.go
    # requests_counter_test.go
)

GO_XTEST_SRCS(
    # xdsclient_test.go
)

END()

RECURSE(
    # e2e_test
    # gotest
    internal
    load
    pool
    tests
    transport
    xdslbregistry
    xdsresource
)
