GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.73.0)

SRCS(
    balancer.go
    blocking_context_dialer.go
    channel.go
    envconfig.go
    http_client.go
    local_listener.go
    marshal_any.go
    parse_port.go
    parse_url.go
    pipe_listener.go
    resolver.go
    restartable_listener.go
    state.go
    status_equal.go
    stubstatshandler.go
    tls_creds.go
    wrappers.go
    wrr.go
    xds_bootstrap.go
)

GO_TEST_SRCS(
    blocking_context_dialer_test.go
    status_equal_test.go
)

GO_XTEST_SRCS(pipe_listener_test.go)

END()

RECURSE(
    fakegrpclb
    gotest
    pickfirst
    proxyserver
    rls
    roundrobin
    stats
    xds
)
