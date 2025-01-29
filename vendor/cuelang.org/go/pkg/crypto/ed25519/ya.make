GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    ed25519.go
    pkg.go
)

GO_XTEST_SRCS(ed25519_test.go)

END()

RECURSE(
    gotest
)
