GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.11.0)

SRCS(
    error.go
    error_post_go120.go
)

GO_TEST_SRCS(
    benchmarks_test.go
    error_post_go120_test.go
    error_test.go
)

GO_XTEST_SRCS(
    appendinvoke_example_test.go
    error_ext_test.go
    example_test.go
)

END()

RECURSE(
    gotest
)
