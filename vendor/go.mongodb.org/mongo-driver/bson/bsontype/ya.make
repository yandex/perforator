GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.17.4)

SRCS(
    bsontype.go
)

GO_TEST_SRCS(bsontype_test.go)

END()

RECURSE(
    gotest
)
