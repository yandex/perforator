GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    hmac.go
    pkg.go
)

GO_XTEST_SRCS(hmac_test.go)

END()

RECURSE(
    gotest
)
