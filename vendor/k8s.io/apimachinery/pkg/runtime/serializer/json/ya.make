GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.34.3)

SRCS(
    collections.go
    json.go
    meta.go
)

GO_TEST_SRCS(
    collections_test.go
    json_limit_test.go
    meta_test.go
)

GO_XTEST_SRCS(json_test.go)

END()

RECURSE(
    gotest
)
