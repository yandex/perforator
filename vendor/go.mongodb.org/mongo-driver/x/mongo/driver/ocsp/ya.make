GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.17.1)

SRCS(
    cache.go
    config.go
    ocsp.go
    options.go
)

GO_TEST_SRCS(
    cache_test.go
    ocsp_test.go
)

END()

RECURSE(
    gotest
)
