GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.2.2)

SRCS(
    oauth2adapt.go
)

GO_TEST_SRCS(oauth2adapt_test.go)

END()

RECURSE(
    gotest
)
