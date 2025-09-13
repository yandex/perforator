GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.73.0)

SRCS(
    encoding.go
    encoding_v2.go
)

GO_XTEST_SRCS(encoding_test.go)

END()

RECURSE(
    gotest
    gzip
    proto
    # yo
)
