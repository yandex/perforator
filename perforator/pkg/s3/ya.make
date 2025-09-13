GO_LIBRARY()

SRCS(
    http_proxy.go
    s3.go
)

GO_TEST_SRCS(http_proxy_test.go)

END()

RECURSE(
    gotest
)
