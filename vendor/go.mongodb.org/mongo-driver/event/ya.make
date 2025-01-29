GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.17.1)

SRCS(
    doc.go
    monitoring.go
)

GO_XTEST_SRCS(examples_test.go)

END()

RECURSE(
    gotest
)
