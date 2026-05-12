GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.0.0-20250604170112-4c0f3b243397)

SRCS(
    ptr.go
)

GO_XTEST_SRCS(ptr_test.go)

END()

RECURSE(
    gotest
)
