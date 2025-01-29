GO_LIBRARY()

LICENSE(MIT)

VERSION(v4.3.0)

SRCS(
    backoff.go
    context.go
    exponential.go
    retry.go
    ticker.go
    timer.go
    tries.go
)

GO_TEST_SRCS(
    backoff_test.go
    context_test.go
    example_test.go
    exponential_test.go
    retry_test.go
    ticker_test.go
    tries_test.go
)

END()

RECURSE(
    gotest
)
