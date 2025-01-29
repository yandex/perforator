GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.0.0-20230726121419-3b25d923346b)

GO_SKIP_TESTS(TestParseIP)

SRCS(
    ip.go
    parse.go
)

GO_TEST_SRCS(ip_test.go)

END()

RECURSE(
    gotest
)
