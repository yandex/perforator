GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.23.2)

SRCS(
    grpcerrors.go
    intercept.go
)

GO_XTEST_SRCS(grpcerrors_test.go)

END()

RECURSE(
    gotest
)
