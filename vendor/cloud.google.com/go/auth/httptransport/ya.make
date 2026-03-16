GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.18.1)

DATA(
    arcadia/vendor/cloud.google.com/go/auth/internal/testdata
)

TEST_CWD(vendor/cloud.google.com/go/auth/httptransport)

SRCS(
    httptransport.go
    transport.go
)

GO_TEST_SRCS(
    httptransport_otel_test.go
    httptransport_test.go
    transport_test.go
)

END()

RECURSE(
    gotest
)
