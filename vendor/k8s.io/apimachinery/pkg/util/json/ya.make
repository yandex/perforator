GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.26.1)

SRCS(
    json.go
)

GO_TEST_SRCS(json_test.go)

END()

RECURSE(
    gotest
)
