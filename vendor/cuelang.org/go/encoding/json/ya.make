GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    json.go
)

GO_TEST_SRCS(json_test.go)

END()

RECURSE(
    gotest
)
