GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v4.0.5)

SRCS(
    asymmetric.go
    crypter.go
    doc.go
    encoding.go
    jwe.go
    jwk.go
    jws.go
    opaque.go
    shared.go
    signing.go
    symmetric.go
)

GO_TEST_SRCS(
    asymmetric_test.go
    crypter_test.go
    doc_test.go
    encoding_test.go
    jwe_test.go
    jwk_test.go
    jws_test.go
    opaque_test.go
    signing_test.go
    symmetric_test.go
    utils_test.go
)

END()

RECURSE(
    cipher
    cryptosigner
    gotest
    json
    jwt
)
