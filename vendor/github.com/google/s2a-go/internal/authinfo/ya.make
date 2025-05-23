GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.1.9)

SRCS(
    authinfo.go
)

GO_TEST_SRCS(authinfo_test.go)

END()

RECURSE(
    gotest
)
