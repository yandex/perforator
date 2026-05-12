GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.33.0)

SRCS(
    cbor.go
    framer.go
    raw.go
)

GO_TEST_SRCS(
    cbor_test.go
    raw_test.go
)

GO_XTEST_SRCS(framer_test.go)

END()

RECURSE(
    direct
    gotest
    internal
)
