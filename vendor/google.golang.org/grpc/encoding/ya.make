GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.79.1)

SRCS(
    encoding.go
    encoding_v2.go
)

GO_XTEST_SRCS(
    compressor_test.go
    encoding_test.go
)

END()

RECURSE(
    gotest
    gzip
    internal
    proto
    # yo
)
