GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.3.0)

SRCS(
    impersonate.go
)

GO_TEST_SRCS(impersonate_test.go)

END()

RECURSE(
    gotest
)
