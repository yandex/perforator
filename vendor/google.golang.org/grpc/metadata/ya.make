GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.73.0)

SRCS(
    metadata.go
)

GO_TEST_SRCS(metadata_test.go)

END()

RECURSE(
    gotest
)
