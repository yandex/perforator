GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.73.0)

SRCS(
    callback_serializer.go
    event.go
    pubsub.go
)

GO_TEST_SRCS(
    callback_serializer_test.go
    event_test.go
    pubsub_test.go
)

END()

RECURSE(
    gotest
)
