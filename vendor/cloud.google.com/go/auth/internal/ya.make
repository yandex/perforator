GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.14.0)

SRCS(
    internal.go
)

GO_TEST_SRCS(internal_test.go)

END()

RECURSE(
    compute
    credsfile
    gotest
    jwt
    testutil
    transport
)
