GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.63.2)

SRCS(
    internal.go
)

GO_TEST_SRCS(internal_test.go)

END()

RECURSE(
    balancer
    clusterspecifier
    gotest
    httpfilter
    resolver
    server
    # test
    testutils
    xdsclient
)
