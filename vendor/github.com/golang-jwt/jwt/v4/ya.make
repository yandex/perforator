GO_LIBRARY()

LICENSE(MIT)

VERSION(v4.5.0)

SRCS(
    claims.go
    doc.go
    ecdsa.go
    ecdsa_utils.go
    ed25519.go
    ed25519_utils.go
    errors.go
    hmac.go
    map_claims.go
    none.go
    parser.go
    parser_option.go
    rsa.go
    rsa_pss.go
    rsa_utils.go
    signing_method.go
    token.go
    types.go
)

GO_TEST_SRCS(
    # map_claims_test.go
)

GO_XTEST_SRCS(
    ecdsa_test.go
    ed25519_test.go
    example_test.go
    hmac_example_test.go
    hmac_test.go
    http_example_test.go
    none_test.go
    parser_test.go
    rsa_pss_test.go
    rsa_test.go
    token_test.go
    types_test.go
)

END()

RECURSE(
    cmd
    gotest
    request
    test
)
