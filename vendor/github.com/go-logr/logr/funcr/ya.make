GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.4.3)

SRCS(
    funcr.go
    slogsink.go
)

GO_TEST_SRCS(
    funcr_test.go
    slogsink_test.go
)

GO_XTEST_SRCS(
    example_formatter_test.go
    example_test.go
)

END()

RECURSE(
    example
    gotest
)
