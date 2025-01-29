GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.46.7)

SRCS(
    api.go
    service.go
    token_provider.go
)

GO_XTEST_SRCS(
    api_test.go
    service_test.go
)

END()

RECURSE(
    gotest
)
