GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v1.6.0)

SRCS(
    murmur.go
    murmur_unsafe.go
)

GO_TEST_SRCS(murmur_test.go)

END()

RECURSE(
    gotest
)
