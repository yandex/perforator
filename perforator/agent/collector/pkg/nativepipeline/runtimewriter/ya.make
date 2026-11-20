GO_LIBRARY()

CGO_SRCS(
    writer.go
)

GO_TEST_SRCS(writer_test.go)

END()

RECURSE_FOR_TESTS(gotest)
