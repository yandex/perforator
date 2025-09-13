GO_LIBRARY()

LICENSE(BSD-2-Clause)

VERSION(v1.14.0)

SRCS(
    bypass.go
    copy.go
    interface.go
)

GO_TEST_SRCS(interface_test.go)

GO_XTEST_SRCS(copy_test.go)

END()

RECURSE(
    gotest
)
