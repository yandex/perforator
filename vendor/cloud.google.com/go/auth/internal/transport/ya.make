GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.3.0)

SRCS(
    cba.go
    s2a.go
    transport.go
)

GO_TEST_SRCS(
    cba_test.go
    s2a_test.go
    transport_test.go
)

END()

RECURSE(
    cert
    gotest
)
