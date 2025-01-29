GO_LIBRARY()

SRCS(
    dummy.go
    inmemory.go
    local.go
    models.go
    remote.go
)

GO_TEST_SRCS(remote_test.go)

END()

RECURSE(
    gotest
)
