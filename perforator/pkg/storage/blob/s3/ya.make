GO_LIBRARY()

SRCS(
    reader.go
    s3.go
)

GO_TEST_SRCS(
    fetchpart_test.go
    reader_test.go
)

END()

RECURSE(
    gotest
)
