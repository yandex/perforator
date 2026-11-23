GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.0.0-20260626114624-be93311217bd)

GO_SKIP_TESTS(TestParseIP)

SRCS(
    ip.go
    parse.go
)

END()
