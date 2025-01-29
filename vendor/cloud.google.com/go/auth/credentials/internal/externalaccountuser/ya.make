GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.3.0)

SRCS(
    externalaccountuser.go
)

GO_TEST_SRCS(externalaccountuser_test.go)

END()

RECURSE(
    gotest
)
