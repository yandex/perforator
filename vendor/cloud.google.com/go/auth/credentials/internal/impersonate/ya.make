GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.15.0)

SRCS(
    idtoken.go
    impersonate.go
)

GO_TEST_SRCS(impersonate_test.go)

END()

RECURSE(
    gotest
)
