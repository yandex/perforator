GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.0.0-20251002143259-bc988d571ff4)

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
