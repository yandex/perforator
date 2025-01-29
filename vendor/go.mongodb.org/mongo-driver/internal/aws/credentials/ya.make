GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.17.1)

SRCS(
    chain_provider.go
    credentials.go
)

GO_TEST_SRCS(
    chain_provider_test.go
    credentials_test.go
)

END()

RECURSE(
    gotest
)
