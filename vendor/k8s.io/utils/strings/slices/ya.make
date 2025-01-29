GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.0.0-20230726121419-3b25d923346b)

SRCS(
    slices.go
)

GO_TEST_SRCS(slices_test.go)

END()

RECURSE(
    gotest
)
