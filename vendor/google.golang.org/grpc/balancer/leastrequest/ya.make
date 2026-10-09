GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.83.2)

SRCS(
    leastrequest.go
)

GO_TEST_SRCS(leastrequest_test.go)

END()

RECURSE(
    gotest
)
