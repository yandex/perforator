GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.37.2)

ALL_GO_TEST_SRCS()

SRCS(
    decimal_int.go
    dns.go
    errors.go
    identifier.go
    kube.go
    path.go
)

END()

RECURSE(
    gotest
)
