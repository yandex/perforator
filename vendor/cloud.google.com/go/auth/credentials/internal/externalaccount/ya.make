GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.3.0)

SRCS(
    aws_provider.go
    executable_provider.go
    externalaccount.go
    file_provider.go
    info.go
    programmatic_provider.go
    url_provider.go
)

GO_TEST_SRCS(
    aws_provider_test.go
    # executable_provider_test.go
    externalaccount_test.go
    file_provider_test.go
    impersonate_test.go
    info_test.go
    programmatic_provider_test.go
    url_provider_test.go
)

END()

RECURSE(
    gotest
)
