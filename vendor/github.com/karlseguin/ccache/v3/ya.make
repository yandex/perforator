GO_LIBRARY()

LICENSE(MIT)

VERSION(v3.0.3)

SRCS(
    bucket.go
    cache.go
    configuration.go
    control.go
    item.go
    layeredbucket.go
    layeredcache.go
    list.go
    secondarycache.go
)

GO_TEST_SRCS(
    bucket_test.go
    cache_test.go
    configuration_test.go
    item_test.go
    layeredcache_test.go
    list_test.go
    secondarycache_test.go
)

END()

RECURSE(
    assert
    gotest
)
