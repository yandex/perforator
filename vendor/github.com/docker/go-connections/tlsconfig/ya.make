GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.5.0)

SRCS(
    certpool.go
    config.go
    config_client_ciphers.go
)

GO_TEST_SRCS(config_test.go)

END()

RECURSE(
    gotest
)
