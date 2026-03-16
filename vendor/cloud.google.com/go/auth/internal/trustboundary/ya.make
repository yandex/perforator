GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.18.1)

SRCS(
    external_accounts_config_providers.go
    trust_boundary.go
)

GO_TEST_SRCS(
    external_accounts_config_providers_test.go
    trust_boundary_test.go
)

END()

RECURSE(
    gotest
)
