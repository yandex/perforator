GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    doc.go
    file.go
    pkg.go
)

GO_TEST_SRCS(file_test.go)

END()

RECURSE(
    gotest
)
