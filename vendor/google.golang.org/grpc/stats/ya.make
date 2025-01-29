GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.63.2)

SRCS(
    handlers.go
    stats.go
)

GO_XTEST_SRCS(stats_test.go)

END()

RECURSE(
    gotest
    # yo
)
