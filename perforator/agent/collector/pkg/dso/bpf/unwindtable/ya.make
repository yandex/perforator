GO_LIBRARY()

SRCS(
    cache.go
    freelist.go
    manager.go
    row.go
)

GO_TEST_SRCS(manager_test.go)

END()

RECURSE_FOR_TESTS(gotest)
