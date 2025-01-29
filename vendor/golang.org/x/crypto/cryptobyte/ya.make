GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.31.0)

SRCS(
    asn1.go
    builder.go
    string.go
)

GO_TEST_SRCS(
    asn1_test.go
    cryptobyte_test.go
)

GO_XTEST_SRCS(example_test.go)

END()

RECURSE(
    asn1
    gotest
)
