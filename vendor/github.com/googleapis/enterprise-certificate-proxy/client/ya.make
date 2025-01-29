GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.3.2)

SRCS(
    client.go
)

GO_TEST_SRCS(
    # client_test.go
)

END()

RECURSE(
    gotest
    util
)
