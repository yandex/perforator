GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.73.0)

SRCS(
    authinfo.go
)

GO_TEST_SRCS(authinfo_test.go)

END()

RECURSE(
    gotest
)
