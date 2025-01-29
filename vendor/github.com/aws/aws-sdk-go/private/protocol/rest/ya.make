GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.46.7)

SRCS(
    build.go
    payload.go
    unmarshal.go
)

GO_TEST_SRCS(
    build_test.go
    unmarshal_test.go
)

GO_XTEST_SRCS(rest_test.go)

END()

RECURSE(
    # gotest
)
