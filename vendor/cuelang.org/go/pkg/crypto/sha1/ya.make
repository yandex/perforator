GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    pkg.go
    sha1.go
)

GO_XTEST_SRCS(sha1_test.go)

END()

RECURSE(
    gotest
)
