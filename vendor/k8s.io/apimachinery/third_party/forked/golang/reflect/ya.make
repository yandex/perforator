GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.33.0)

SRCS(
    deep_equal.go
)

GO_TEST_SRCS(deep_equal_test.go)

END()

RECURSE(
    gotest
)
