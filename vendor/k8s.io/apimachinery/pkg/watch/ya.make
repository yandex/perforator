GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.26.1)

SRCS(
    doc.go
    filter.go
    mux.go
    streamwatcher.go
    watch.go
    zz_generated.deepcopy.go
)

GO_TEST_SRCS(mux_test.go)

GO_XTEST_SRCS(
    filter_test.go
    streamwatcher_test.go
    watch_test.go
)

END()

RECURSE(
    gotest
)
