GO_LIBRARY()

SRCS(
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
