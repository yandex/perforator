GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.83.2)

SRCS(
    balancergroup.go
    balancerstateaggregator.go
)

GO_XTEST_SRCS(balancergroup_test.go)

END()

RECURSE(
    gotest
)
