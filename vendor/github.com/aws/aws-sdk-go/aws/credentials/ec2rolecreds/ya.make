GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.46.7)

SRCS(
    ec2_role_provider.go
)

GO_XTEST_SRCS(ec2_role_provider_test.go)

END()

RECURSE(
    gotest
)
