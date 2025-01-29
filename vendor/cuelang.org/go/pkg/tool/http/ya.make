GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    doc.go
    http.go
    pkg.go
)

GO_TEST_SRCS(http_test.go)

END()

RECURSE(
    gotest
)
