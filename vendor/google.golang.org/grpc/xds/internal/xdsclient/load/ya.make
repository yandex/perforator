GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.73.0)

SRCS(
    reporter.go
    store.go
)

GO_TEST_SRCS(store_test.go)

END()

RECURSE(
    gotest
)
