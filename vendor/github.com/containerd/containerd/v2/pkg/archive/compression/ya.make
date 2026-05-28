GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v2.1.5)

SRCS(
    compression.go
)

GO_TEST_SRCS(
    # benchmark_test.go
    compression_fuzzer_test.go
    # compression_test.go
)

END()

RECURSE(
    gotest
)
