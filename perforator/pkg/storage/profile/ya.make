GO_LIBRARY()

PEERDIR(perforator/pkg/cprofile)

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
