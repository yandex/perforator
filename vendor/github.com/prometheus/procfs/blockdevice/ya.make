GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.21.1)

SRCS(
    dm_multipath.go
    stats.go
)

GO_TEST_SRCS(
    dm_multipath_test.go
    stats_test.go
)

END()

RECURSE(
    # gotest
)
