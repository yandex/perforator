GO_LIBRARY()

LICENSE(MPL-2.0)

VERSION(v1.12.2)

SRCS(
    slices.go
)

GO_TEST_SRCS(slices_test.go)

END()

RECURSE(
    gotest
)
