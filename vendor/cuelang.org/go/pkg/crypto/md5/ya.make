GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    md5.go
    pkg.go
)

GO_XTEST_SRCS(md5_test.go)

END()

RECURSE(
    gotest
)
