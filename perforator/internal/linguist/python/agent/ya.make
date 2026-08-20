GO_LIBRARY()

SRCS(
    stackprocessor.go
    symbolizer.go
)

GO_TEST_SRCS(
    stackprocessor_test.go
    symbolizer_test.go
)

END()

RECURSE(
    linetable
    remotemem
)

RECURSE_FOR_TESTS(
    gotest
)
