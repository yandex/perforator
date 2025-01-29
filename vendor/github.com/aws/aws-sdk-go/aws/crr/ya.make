GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.46.7)

SRCS(
    cache.go
    endpoint.go
    sync_map.go
)

GO_TEST_SRCS(
    cache_test.go
    endpoint_test.go
    sync_map_test.go
)

END()

RECURSE(
    gotest
)
