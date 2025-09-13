GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.4.3)

SRCS(
    context.go
    context_slog.go
    discard.go
    logr.go
    sloghandler.go
    slogr.go
    slogsink.go
)

GO_TEST_SRCS(
    context_slog_test.go
    context_test.go
    discard_test.go
    logr_test.go
    slogr_test.go
    testimpls_slog_test.go
    testimpls_test.go
)

GO_XTEST_SRCS(
    example_marshaler_secret_test.go
    example_marshaler_test.go
    example_slogr_test.go
    example_test.go
)

END()

RECURSE(
    benchmark
    examples
    funcr
    # gotest
    internal
    slogr
    testing
    testr
)
