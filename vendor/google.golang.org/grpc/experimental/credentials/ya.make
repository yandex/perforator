GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.73.0)

DATA(
    arcadia/vendor/google.golang.org/grpc/testdata/x509
)

TEST_CWD(vendor/google.golang.org/grpc/experimental/credentials)

SRCS(
    tls.go
)

GO_TEST_SRCS(credentials_test.go)

GO_XTEST_SRCS(
    # tls_ext_test.go
)

END()

RECURSE(
    gotest
    internal
)
