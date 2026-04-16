GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.79.3)

SRCS(
    endpointsharding.go
)

GO_TEST_SRCS(
    # endpointsharding_test.go
)

GO_XTEST_SRCS(
    # endpointsharding_ext_test.go
)

END()

RECURSE(
    gotest
)
