GO_LIBRARY()

SRCS(
    event.go
    namecache.go
    tracker.go
)

GO_TEST_SRCS(tracker_test.go)

END()

RECURSE(
    gotest
)
