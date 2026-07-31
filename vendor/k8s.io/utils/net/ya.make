GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.0.0-20251002143259-bc988d571ff4)

SRCS(
    ipfamily.go
    ipnet.go
    multi_listen.go
    net.go
    parse.go
    port.go
)

GO_TEST_SRCS(
    ipfamily_test.go
    ipnet_test.go
    ips_test.go
    multi_listen_test.go
    net_test.go
    parse_test.go
    port_test.go
)

END()

RECURSE(
    gotest
)
