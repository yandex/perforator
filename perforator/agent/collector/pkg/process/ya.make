GO_LIBRARY()

SRCS(
    buildid_cache.go
    listener.go
    map.go
    scanner.go
)

GO_TEST_SRCS(
    process_info_test.go
)

END()

RECURSE(
    gotest
)
