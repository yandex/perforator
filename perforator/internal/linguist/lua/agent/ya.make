GO_LIBRARY()

SRCS(
    lua.go
    stackprocessor.go
)

GO_TEST_SRCS(stackprocessor_test.go)

END()

RECURSE_FOR_TESTS(gotest)
