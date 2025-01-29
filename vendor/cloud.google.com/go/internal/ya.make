GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.112.2)

SRCS(
    annotate.go
    retry.go
)

GO_TEST_SRCS(
    annotate_test.go
    retry_test.go
)

END()

RECURSE(
    btree
    detect
    fields
    gotest
    leakcheck
    optional
    pretty
    protostruct
    pubsub
    testutil
    trace
    tracecontext
    uid
    version
)
