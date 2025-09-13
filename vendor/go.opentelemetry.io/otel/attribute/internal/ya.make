GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.37.0)

SRCS(
    attribute.go
)

GO_TEST_SRCS(attribute_test.go)

END()

RECURSE(
    gotest
)
