GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.1.7)

SRCS(
    s2a_fallback.go
)

GO_TEST_SRCS(s2a_fallback_test.go)

END()

RECURSE(
    gotest
)
