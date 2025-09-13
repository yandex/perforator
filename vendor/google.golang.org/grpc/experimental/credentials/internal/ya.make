GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.73.0)

DATA(
    arcadia/vendor/google.golang.org/grpc/testdata/x509
)

TEST_CWD(vendor/google.golang.org/grpc/experimental/credentials/internal)

SRCS(
    spiffe.go
    syscallconn.go
)

GO_TEST_SRCS(
    spiffe_test.go
    syscallconn_test.go
)

END()

RECURSE(
    gotest
)
