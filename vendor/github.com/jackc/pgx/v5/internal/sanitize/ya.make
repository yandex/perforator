GO_LIBRARY()

LICENSE(MIT)

VERSION(v5.7.1)

SRCS(
    sanitize.go
)

GO_XTEST_SRCS(sanitize_test.go)

END()

RECURSE(
    gotest
)
