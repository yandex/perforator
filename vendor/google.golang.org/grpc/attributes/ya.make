GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.73.0)

SRCS(
    attributes.go
)

GO_XTEST_SRCS(attributes_test.go)

END()

RECURSE(
    gotest
)
