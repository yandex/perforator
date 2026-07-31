GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.0.0-20251002143259-bc988d571ff4)

SRCS(
    ptr.go
)

GO_XTEST_SRCS(ptr_test.go)

END()

RECURSE(
    gotest
)
