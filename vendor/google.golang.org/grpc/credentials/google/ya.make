GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.83.2)

SRCS(
    gcp_service_account_identity_credentials.go
    google.go
    xds.go
)

GO_TEST_SRCS(
    google_test.go
    xds_test.go
)

END()

RECURSE(
    gotest
    internal
)
