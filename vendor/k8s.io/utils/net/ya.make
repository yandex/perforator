GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.0.0-20230726121419-3b25d923346b)

SRCS(
    ipfamily.go
    ipnet.go
    net.go
    parse.go
    port.go
)

GO_TEST_SRCS(
    ipfamily_test.go
    ipnet_test.go
    net_test.go
    port_test.go
)

END()

RECURSE(
    gotest
)
