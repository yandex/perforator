GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.17.1)

SRCS(
    auth.go
    aws_conv.go
    conversation.go
    cred.go
    default.go
    doc.go
    gssapi_not_enabled.go
    mongodbaws.go
    mongodbcr.go
    oidc.go
    plain.go
    sasl.go
    scram.go
    util.go
    x509.go
)

GO_TEST_SRCS(
    # mongodbaws_test.go
    # oidc_test.go
    # scram_test.go
    # speculative_scram_test.go
    # speculative_x509_test.go
)

GO_XTEST_SRCS(
    auth_spec_test.go
    auth_test.go
    mongodbcr_test.go
    plain_test.go
)

END()

RECURSE(
    creds
    gotest
)
