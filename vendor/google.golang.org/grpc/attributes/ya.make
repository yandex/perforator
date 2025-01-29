GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.63.2)

SRCS(
    attributes.go
)

GO_XTEST_SRCS(attributes_test.go)

END()

RECURSE(
    gotest
)
