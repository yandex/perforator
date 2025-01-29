GO_LIBRARY()

SRCS(
    cached_client.go
    client.go
)

GO_XTEST_SRCS(client_test.go)

END()

RECURSE(
    gotest
)
