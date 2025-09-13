GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.73.0)

SRCS(
    dns_resolver.go
)

GO_XTEST_SRCS(
    dns_resolver_test.go
    fake_net_resolver_test.go
)

END()

RECURSE(
    gotest
    internal
)
