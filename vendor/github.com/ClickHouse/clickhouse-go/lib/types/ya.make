GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.5.4)

SRCS(
    date.go
    uuid.go
)

GO_TEST_SRCS(uuid_test.go)

END()

RECURSE(
    gotest
)
