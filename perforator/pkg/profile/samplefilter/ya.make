GO_LIBRARY()

SRCS(
    buildid_filter.go
    env_filter.go
    filter.go
    tls_filter.go
)

GO_TEST_SRCS(
    buildid_filter_test.go
    env_filter_test.go
    tls_filter_test.go
)

END()

RECURSE(
    gotest
)
