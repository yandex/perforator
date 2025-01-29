GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.1.7)

SRCS(
    iam.go
)

GO_TEST_SRCS(iam_test.go)

END()

RECURSE(
    admin
    apiv1
    apiv2
    credentials
    gotest
    internal
)
