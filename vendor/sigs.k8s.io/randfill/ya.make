GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.0.0)

SRCS(
    randfill.go
)

GO_TEST_SRCS(randfill_test.go)

GO_XTEST_SRCS(example_test.go)

END()

RECURSE(
    bytesource
    gotest
)
