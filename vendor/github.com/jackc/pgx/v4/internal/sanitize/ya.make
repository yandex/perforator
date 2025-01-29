GO_LIBRARY()

LICENSE(MIT)

VERSION(v4.18.3)

SRCS(
    sanitize.go
)

GO_XTEST_SRCS(sanitize_test.go)

END()

RECURSE(
    gotest
)
