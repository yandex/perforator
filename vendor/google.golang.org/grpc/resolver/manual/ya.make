GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.73.0)

SRCS(
    manual.go
)

GO_XTEST_SRCS(manual_test.go)

END()

RECURSE(
    gotest
)
