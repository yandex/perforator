GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.0.0-20241104100929-3ea5e8cea738)

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
    multi_listen_test.go
    net_test.go
    port_test.go
)

END()

RECURSE(
    gotest
)
