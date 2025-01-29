GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.46.7)

SRCS(
    credentials.go
    custom_transport.go
    doc.go
    env_config.go
    session.go
    shared_config.go
)

GO_TEST_SRCS(
    client_tls_cert_test.go
    credentials_test.go
    csm_test.go
    custom_ca_bundle_test.go
    env_config_test.go
    session_test.go
    shared_config_test.go
    shared_test.go
)

END()

RECURSE(
    gotest
)
