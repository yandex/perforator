GO_LIBRARY()

SRCS(
    reader_linux.go
)

GO_TEST_SRCS(reader_linux_test.go)

END()

RECURSE_FOR_TESTS(
    gotest
)
