GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v2.2.0)

SRCS(
    reference.go
)

GO_TEST_SRCS(reference_test.go)

END()

RECURSE(
    gotest
)
