GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    manual.go
    pkg.go
    strconv.go
)

GO_XTEST_SRCS(strconv_test.go)

END()

RECURSE(
    gotest
)
