GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.73.0)

SRCS(
    carrier.go
)

GO_TEST_SRCS(carrier_test.go)

END()

RECURSE(
    gotest
)
