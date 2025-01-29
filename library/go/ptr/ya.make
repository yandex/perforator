GO_LIBRARY()

SRCS(
    ptr.go
)

GO_TEST_SRCS(ptr_test.go)

END()

RECURSE(
    gotest
)
