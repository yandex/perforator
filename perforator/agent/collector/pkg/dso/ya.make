GO_LIBRARY()

SRCS(
    map.go
    storage.go
)

GO_TEST_SRCS(storage_test.go)

END()

RECURSE(
    bpf
    gotest
    parser
)
