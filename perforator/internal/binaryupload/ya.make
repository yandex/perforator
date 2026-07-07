GO_LIBRARY()

SRCS(
    binaryupload.go
)

GO_TEST_SRCS(
    binaryupload_test.go
)

END()

RECURSE(
    gotest
)
