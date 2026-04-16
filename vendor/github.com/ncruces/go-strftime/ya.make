GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.0.0)

SRCS(
    parser.go
    pkg.go
    specifiers.go
    strftime.go
)

GO_TEST_SRCS(parser_test.go)

GO_XTEST_SRCS(
    bench_test.go
    example_test.go
    external_test.go
    fuzz_test.go
    strftime_test.go
)

END()

RECURSE(
    gotest
)
