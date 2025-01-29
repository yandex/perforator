GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v2.120.1)

SRCS(
    keyvalues.go
    keyvalues_slog.go
)

GO_XTEST_SRCS(keyvalues_test.go)

END()

RECURSE(
    gotest
)
