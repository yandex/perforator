GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.18.2)

SRCS(
    internal.go
    version.go
)

GO_TEST_SRCS(internal_test.go)

END()

RECURSE(
    compute
    credsfile
    gotest
    jwt
    retry
    testutil
    transport
    trustboundary
)
