GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.24.0)

SRCS(
    b3.go
)

GO_TEST_SRCS(b3_test.go)

END()

RECURSE(
    gotest
)
