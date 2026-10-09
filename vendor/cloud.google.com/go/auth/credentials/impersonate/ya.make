GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.18.2)

SRCS(
    doc.go
    idtoken.go
    impersonate.go
    user.go
)

GO_TEST_SRCS(
    idtoken_test.go
    impersonate_test.go
    user_test.go
)

GO_XTEST_SRCS(
    # example_test.go
    # integration_test.go
)

END()

RECURSE(
    gotest
)
