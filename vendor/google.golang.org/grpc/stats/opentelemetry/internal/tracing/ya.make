GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.79.1)

SRCS(
    carrier.go
)

GO_TEST_SRCS(carrier_test.go)

END()

RECURSE(
    gotest
)
