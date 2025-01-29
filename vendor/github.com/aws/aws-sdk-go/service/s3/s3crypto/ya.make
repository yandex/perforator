GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.46.7)

SRCS(
    aes_cbc.go
    aes_cbc_content_cipher.go
    aes_cbc_padder.go
    aes_gcm.go
    aes_gcm_content_cipher.go
    cipher.go
    cipher_builder.go
    cipher_util.go
    crypto_registry.go
    decryption_client.go
    decryption_client_v2.go
    doc.go
    encryption_client.go
    encryption_client_v2.go
    envelope.go
    errors.go
    fixture.go
    hash_io.go
    helper.go
    key_handler.go
    kms_context_key_handler.go
    kms_key_handler.go
    mat_desc.go
    padder.go
    pkcs7_padder.go
    shared_client.go
    strategy.go
)

GO_TEST_SRCS(
    aes_cbc_content_cipher_test.go
    aes_cbc_padder_test.go
    aes_cbc_test.go
    aes_gcm_content_cipher_test.go
    aes_gcm_test.go
    cipher_util_test.go
    crypto_registry_test.go
    encryption_client_test.go
    encryption_client_v2_test.go
    envelope_test.go
    hash_io_test.go
    helper_test.go
    key_handler_test.go
    kms_context_key_handler_test.go
    kms_key_handler_test.go
    mat_desc_test.go
    mock_test.go
)

GO_XTEST_SRCS(
    cipher_test.go
    decryption_client_test.go
    decryption_client_v2_test.go
    migrations_test.go
    pkcs7_padder_test.go
    strategy_test.go
)

END()

RECURSE(
    gotest
)
