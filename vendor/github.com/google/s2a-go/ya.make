GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.1.7)

SRCS(
    s2a.go
    s2a_options.go
    s2a_utils.go
)

GO_TEST_SRCS(
    s2a_e2e_test.go
    s2a_options_test.go
    s2a_test.go
    s2a_utils_test.go
)

GO_TEST_EMBED_PATTERN(testdata/client_cert.pem)

GO_TEST_EMBED_PATTERN(testdata/client_key.pem)

GO_TEST_EMBED_PATTERN(testdata/mds_client_cert.pem)

GO_TEST_EMBED_PATTERN(testdata/mds_client_key.pem)

GO_TEST_EMBED_PATTERN(testdata/mds_root_cert.pem)

GO_TEST_EMBED_PATTERN(testdata/mds_server_cert.pem)

GO_TEST_EMBED_PATTERN(testdata/mds_server_key.pem)

GO_TEST_EMBED_PATTERN(testdata/self_signed_cert.pem)

GO_TEST_EMBED_PATTERN(testdata/self_signed_key.pem)

GO_TEST_EMBED_PATTERN(testdata/server_cert.pem)

GO_TEST_EMBED_PATTERN(testdata/server_key.pem)

END()

RECURSE(
    fallback
    gotest
    internal
    retry
    stream
)
