GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.14.0)

SRCS(
    context_watcher.go
)

GO_XTEST_SRCS(context_watcher_test.go)

END()

RECURSE(
    gotest
)
