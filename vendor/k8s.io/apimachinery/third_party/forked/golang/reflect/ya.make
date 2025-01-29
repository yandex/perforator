GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.26.1)

SRCS(
    deep_equal.go
)

GO_TEST_SRCS(deep_equal_test.go)

END()

RECURSE(
    gotest
)
