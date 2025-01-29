GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.31.0)

SRCS(
    bmp-string.go
    crypto.go
    errors.go
    mac.go
    pbkdf.go
    pkcs12.go
    safebags.go
)

GO_TEST_SRCS(
    bmp-string_test.go
    crypto_test.go
    mac_test.go
    pbkdf_test.go
    pkcs12_test.go
)

END()

RECURSE(
    gotest
    internal
)
