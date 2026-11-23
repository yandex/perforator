GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.0.0)

SRCS(
    reference.go
)

GO_TEST_SRCS(
    fuzz_test.go
    reference_test.go
)

GO_XTEST_SRCS(examples_test.go)

END()

RECURSE(
    gotest
    internal
)
