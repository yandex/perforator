GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.0.0-20240726163527-a2c0da244d78)

SRCS(
    cipher.go
    cipher_3des.go
    cipher_aes.go
    kdf_pbkdf2.go
    kdf_scrypt.go
    pkcs8.go
)

GO_XTEST_SRCS(pkcs8_test.go)

END()

RECURSE(
    gotest
)
