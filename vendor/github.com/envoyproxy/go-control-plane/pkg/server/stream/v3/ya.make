GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.14.0)

SRCS(
    stream.go
    subscription.go
)

GO_TEST_SRCS(subscription_test.go)

END()

RECURSE(
    gotest
)
