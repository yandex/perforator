GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.3.5)

SRCS(
    types.go
)

GO_TEST_SRCS(types_test.go)

END()

RECURSE(
    gotest
)
