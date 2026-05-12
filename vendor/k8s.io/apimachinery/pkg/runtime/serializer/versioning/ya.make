GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.33.0)

SRCS(
    versioning.go
)

GO_TEST_SRCS(
    versioning_test.go
    versioning_unstructured_test.go
)

END()

RECURSE(
    gotest
)
