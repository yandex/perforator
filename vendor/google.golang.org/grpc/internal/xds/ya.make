GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.79.3)

SRCS(
    xds.go
)

GO_TEST_SRCS(xds_test.go)

END()

RECURSE(
    balancer
    bootstrap
    clients
    clusterspecifier
    gotest
    httpfilter
    matcher
    rbac
    resolver
    server
    test
    testutils
    xdsclient
    xdsdepmgr
)
