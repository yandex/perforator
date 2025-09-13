GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.7.0)

SRCS(
    murmur.go
    murmur_unsafe.go
)

GO_TEST_SRCS(murmur_test.go)

END()

RECURSE(
    gotest
)
