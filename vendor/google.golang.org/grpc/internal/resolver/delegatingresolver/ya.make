GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.79.3)

SRCS(
    delegatingresolver.go
)

GO_TEST_SRCS(
    # delegatingresolver_test.go
)

GO_XTEST_SRCS(delegatingresolver_ext_test.go)

END()

RECURSE(
    gotest
)
