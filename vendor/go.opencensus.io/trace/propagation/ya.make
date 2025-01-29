GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.24.0)

SRCS(
    propagation.go
)

GO_TEST_SRCS(propagation_test.go)

END()

RECURSE(
    gotest
)
