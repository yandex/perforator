GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.73.0)

SRCS(
    ads_stream.go
    authority.go
    channel.go
    clientimpl_watchers.go
    logging.go
    resource_type.go
    resource_watcher.go
    xdsclient.go
    xdsconfig.go
)

GO_TEST_SRCS(
    channel_test.go
    helpers_test.go
    xdsclient_test.go
)

END()

RECURSE(
    gotest
    internal
    metrics
    test
)
