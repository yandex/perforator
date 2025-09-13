GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.0.0-20241104100929-3ea5e8cea738)

SRCS(
    slices.go
)

GO_TEST_SRCS(slices_test.go)

END()

RECURSE(
    gotest
)
