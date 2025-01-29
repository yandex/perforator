GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.3.2)

SRCS(
    util.go
)

GO_TEST_SRCS(
    # util_test.go
)

END()

RECURSE(
    gotest
)
