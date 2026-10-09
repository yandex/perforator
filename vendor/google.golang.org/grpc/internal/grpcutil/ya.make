GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.83.2)

SRCS(
    compressor.go
    encode_duration.go
    grpcutil.go
    metadata.go
    method.go
)

GO_TEST_SRCS(
    compressor_test.go
    encode_duration_test.go
    method_test.go
)

END()

RECURSE(
    gotest
)
