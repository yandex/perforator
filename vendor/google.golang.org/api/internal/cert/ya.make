GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.176.1)

SRCS(
    default_cert.go
    enterprise_cert.go
    secureconnect_cert.go
)

GO_TEST_SRCS(
    # enterprise_cert_test.go
    # secureconnect_cert_test.go
)

END()

RECURSE(
    gotest
)
