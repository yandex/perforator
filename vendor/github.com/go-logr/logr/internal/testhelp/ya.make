GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.4.3)

SRCS(
    slog.go
)

GO_TEST_SRCS(slog_test.go)

END()

RECURSE(
    gotest
)
