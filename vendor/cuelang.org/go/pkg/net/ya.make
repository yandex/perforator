GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    doc.go
    host.go
    ip.go
    pkg.go
)

GO_XTEST_SRCS(net_test.go)

END()

RECURSE(
    gotest
)
