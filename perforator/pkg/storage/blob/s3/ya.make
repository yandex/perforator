GO_LIBRARY()

SRCS(
    download.go
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
