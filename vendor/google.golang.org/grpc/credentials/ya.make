GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.79.1)

SRCS(
    credentials.go
    tls.go
)

GO_TEST_SRCS(credentials_test.go)

GO_XTEST_SRCS(
    credentials_ext_test.go
    tls_ext_test.go
)

END()

RECURSE(
    alts
    google
    # gotest
    insecure
    jwt
    local
    oauth
    sts
    tls
    xds
)
