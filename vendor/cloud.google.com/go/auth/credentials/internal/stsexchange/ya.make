GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.3.0)

SRCS(
    sts_exchange.go
)

GO_TEST_SRCS(sts_exchange_test.go)

END()

RECURSE(
    gotest
)
