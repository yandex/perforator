GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.0.0-20260210185600-b8788abfbbc2)

GO_SKIP_TESTS(TestParseIP)

SRCS(
    ip.go
    parse.go
)

END()
