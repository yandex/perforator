GO_TEST()

LICENSE(Apache-2.0)

VERSION(v1.73.0)

DATA(
    arcadia/vendor/google.golang.org/grpc/testdata/x509
)

TEST_CWD(vendor/google.golang.org/grpc/xds/test)

GO_XTEST_SRCS(eds_resource_missing_test.go)

END()
