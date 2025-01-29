GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.17.1)

SRCS(
    uuid.go
)

GO_TEST_SRCS(uuid_test.go)

END()

RECURSE(
    gotest
)
