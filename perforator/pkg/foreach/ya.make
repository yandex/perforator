GO_LIBRARY()

SRCS(
    foreach.go
)

GO_TEST_SRCS(foreach_test.go)

END()

RECURSE(
    gotest
)
