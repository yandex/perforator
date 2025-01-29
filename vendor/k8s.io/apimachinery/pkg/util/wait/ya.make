GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.26.1)

SRCS(
    doc.go
    wait.go
)

GO_TEST_SRCS(wait_test.go)

END()

RECURSE(
    gotest
)
