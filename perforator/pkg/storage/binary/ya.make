GO_LIBRARY()

SRCS(
    config.go
    storage.go
)

GO_TEST_SRCS(
    storage_test.go
)

END()

RECURSE(
    gotest
    meta
)
