GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.63.2)

SRCS(
    credentials.go
    tls.go
)

GO_TEST_SRCS(credentials_test.go)

GO_XTEST_SRCS(tls_ext_test.go)

END()

RECURSE(
    alts
    google
    # gotest
    insecure
    local
    oauth
    sts
    tls
    xds
)
