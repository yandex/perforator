GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.1.7)

SRCS(
    s2av2.go
)

GO_TEST_SRCS(
    s2av2_e2e_test.go
    s2av2_test.go
)

GO_TEST_EMBED_PATTERN(testdata/client_cert.pem)

GO_TEST_EMBED_PATTERN(testdata/client_key.pem)

GO_TEST_EMBED_PATTERN(testdata/server_cert.pem)

GO_TEST_EMBED_PATTERN(testdata/server_key.pem)

END()

RECURSE(
    certverifier
    fakes2av2
    gotest
    remotesigner
    tlsconfigstore
)
