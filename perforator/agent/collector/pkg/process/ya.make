GO_LIBRARY()

SRCS(
    buildid_cache.go
    listener.go
    map.go
    pidns_index.go
    scanner.go
)

GO_TEST_SRCS(
    map_test.go
    process_info_test.go
)

END()

RECURSE(
    gotest
)
