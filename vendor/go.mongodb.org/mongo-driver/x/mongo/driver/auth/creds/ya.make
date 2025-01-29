GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.17.1)

SRCS(
    awscreds.go
    azurecreds.go
    doc.go
    gcpcreds.go
)

GO_TEST_SRCS(credscaching_test.go)

END()

RECURSE(
    gotest
)
