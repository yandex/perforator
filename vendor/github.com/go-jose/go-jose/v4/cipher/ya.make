GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v4.0.5)

SRCS(
    cbc_hmac.go
    concat_kdf.go
    ecdh_es.go
    key_wrap.go
)

GO_TEST_SRCS(
    cbc_hmac_test.go
    concat_kdf_test.go
    ecdh_es_test.go
    key_wrap_test.go
)

END()

RECURSE(
    gotest
)
