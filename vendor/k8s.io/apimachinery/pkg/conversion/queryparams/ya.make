GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.26.1)

SRCS(
    convert.go
    doc.go
)

GO_XTEST_SRCS(convert_test.go)

END()

RECURSE(
    gotest
)
