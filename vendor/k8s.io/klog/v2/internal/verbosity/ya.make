GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v2.120.1)

SRCS(
    verbosity.go
)

GO_TEST_SRCS(
    helper_test.go
    verbosity_test.go
)

END()

RECURSE(
    gotest
)
