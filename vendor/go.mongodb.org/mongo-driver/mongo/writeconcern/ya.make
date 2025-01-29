GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.17.1)

SRCS(
    writeconcern.go
)

GO_XTEST_SRCS(
    writeconcern_example_test.go
    writeconcern_test.go
)

END()

RECURSE(
    gotest
)
