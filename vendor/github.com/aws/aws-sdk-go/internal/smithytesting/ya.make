GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.46.7)

SRCS(
    document.go
)

GO_TEST_SRCS(document_test.go)

END()

RECURSE(
    gotest
    xml
)
