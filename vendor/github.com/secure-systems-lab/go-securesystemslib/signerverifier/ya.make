GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.9.0)

SRCS(
    ecdsa.go
    ed25519.go
    rsa.go
    signerverifier.go
    utils.go
)

GO_TEST_SRCS(
    ecdsa_test.go
    ed25519_test.go
    rsa_test.go
    signerverifier_test.go
    utils_test.go
)

GO_TEST_EMBED_PATTERN(test-data/ecdsa-test-key-pem)

GO_TEST_EMBED_PATTERN(test-data/ecdsa-test-key-pem.pub)

GO_TEST_EMBED_PATTERN(test-data/ed25519-test-key-pem)

GO_TEST_EMBED_PATTERN(test-data/ed25519-test-key-pem.pub)

GO_TEST_EMBED_PATTERN(test-data/rsa-test-key)

GO_TEST_EMBED_PATTERN(test-data/rsa-test-key-pkcs8)

GO_TEST_EMBED_PATTERN(test-data/rsa-test-key.pub)

END()

RECURSE(
    gotest
)
