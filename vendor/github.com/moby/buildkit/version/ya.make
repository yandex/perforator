GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.13.2)

SRCS(
    ua.go
    version.go
)

GO_TEST_SRCS(ua_test.go)

END()

RECURSE(
    gotest
)
