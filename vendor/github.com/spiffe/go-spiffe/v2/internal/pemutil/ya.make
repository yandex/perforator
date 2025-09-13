GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v2.5.0)

SRCS(
    pem.go
)

GO_XTEST_SRCS(pem_test.go)

END()

RECURSE(
    gotest
)
