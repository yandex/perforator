GO_LIBRARY()

SRCS(
    pubsub.go
)

GO_TEST_SRCS(pubsub_test.go)

END()

RECURSE(
    gotest
)
