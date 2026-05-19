GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.7.29)

GO_SKIP_TESTS(
    TestCmdStream
    TestCmdStreamBad
)

SRCS(
    compression.go
)

GO_TEST_SRCS(
    benchmark_test.go
    compression_test.go
)

END()

RECURSE(
    gotest
)
