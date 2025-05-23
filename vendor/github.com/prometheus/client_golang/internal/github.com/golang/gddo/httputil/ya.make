GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.21.1)

SRCS(
    negotiate.go
)

GO_TEST_SRCS(negotiate_test.go)

END()

RECURSE(
    gotest
    header
)
