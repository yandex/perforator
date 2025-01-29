GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    scanner.go
)

GO_TEST_SRCS(scanner_test.go)

END()

RECURSE(
    gotest
)
