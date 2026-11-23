GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.37.2)

SRCS(
    cbor.go
    collections.go
    framer.go
    raw.go
)

GO_TEST_SRCS(
    cbor_test.go
    collections_test.go
    raw_test.go
)

GO_XTEST_SRCS(framer_test.go)

END()

RECURSE(
    direct
    gotest
    internal
)
