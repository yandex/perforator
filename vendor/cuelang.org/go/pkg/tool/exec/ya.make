GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    doc.go
    exec.go
    pkg.go
)

GO_TEST_SRCS(exec_test.go)

END()

RECURSE(
    gotest
)
