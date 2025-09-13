GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.2.5)

SRCS(
    stackdump.go
)

GO_XTEST_SRCS(stackdump_test.go)

END()

RECURSE(
    gotest
)
