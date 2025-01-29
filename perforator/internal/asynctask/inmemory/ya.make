GO_LIBRARY()

SRCS(
    config.go
    inmemory.go
)

GO_TEST_SRCS(inmemory_test.go)

END()

RECURSE(
    gotest
)
