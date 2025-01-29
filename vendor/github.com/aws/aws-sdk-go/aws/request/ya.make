GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.46.7)

SRCS(
    connection_reset_error.go
    handlers.go
    http_request.go
    offset_reader.go
    request.go
    request_1_8.go
    request_context.go
    request_pagination.go
    retryer.go
    timeout_read_closer.go
    validation.go
    waiter.go
)

GO_TEST_SRCS(
    http_request_copy_test.go
    offset_reader_test.go
    request_internal_test.go
    request_resetbody_test.go
    request_retry_test.go
    retryer_test.go
    timeout_read_closer_test.go
)

GO_XTEST_SRCS(
    connection_reset_error_test.go
    handlers_test.go
    http_request_retry_test.go
    request_1_6_test.go
    request_1_8_test.go
    request_context_test.go
    request_pagination_test.go
    request_test.go
    timeout_read_closer_benchmark_test.go
    waiter_test.go
)

END()

RECURSE(
    # gotest
)
