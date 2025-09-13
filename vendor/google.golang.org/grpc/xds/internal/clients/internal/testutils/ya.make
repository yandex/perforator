GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.73.0)

SRCS(
    channel.go
    marshal_any.go
    restartable_listener.go
    wrappers.go
)

END()

RECURSE(
    e2e
    fakeserver
)
