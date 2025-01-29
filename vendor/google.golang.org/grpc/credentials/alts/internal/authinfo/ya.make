GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.63.2)

SRCS(
    authinfo.go
)

GO_TEST_SRCS(authinfo_test.go)

END()

RECURSE(
    gotest
)
