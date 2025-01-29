GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    pkg.go
    sha512.go
)

GO_XTEST_SRCS(sha512_test.go)

END()

RECURSE(
    gotest
)
