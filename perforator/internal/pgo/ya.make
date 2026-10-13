GO_LIBRARY()

SRCS(
    llvm.go
)

GO_TEST_SRCS(llvm_test.go)

END()

RECURSE_FOR_TESTS(
    gotest
)
