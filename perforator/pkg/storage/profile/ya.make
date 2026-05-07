GO_LIBRARY()

SRCS(
    config.go
    models.go
    storage.go
)

GO_TEST_SRCS(storage_container_test.go)

END()

RECURSE(
    compound
    gotest
    meta
)
