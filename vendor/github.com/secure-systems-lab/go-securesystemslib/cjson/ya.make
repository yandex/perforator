GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.9.0)

SRCS(
    canonicaljson.go
)

GO_TEST_SRCS(canonicaljson_test.go)

END()

RECURSE(
    gotest
)
