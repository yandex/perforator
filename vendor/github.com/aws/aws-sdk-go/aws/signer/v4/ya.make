GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.46.7)

SRCS(
    header_rules.go
    options.go
    request_context_go1.7.go
    stream.go
    uri_path.go
    v4.go
)

GO_TEST_SRCS(
    header_rules_test.go
    headers_test.go
    stream_test.go
    v4_test.go
)

GO_XTEST_SRCS(
    functional_1_5_test.go
    functional_test.go
)

END()

RECURSE(
    # gotest
)
