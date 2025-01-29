GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.63.2)

SRCS(
    csds.go
)

GO_XTEST_SRCS(
    # csds_e2e_test.go
)

END()

RECURSE(
    gotest
)
