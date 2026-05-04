GO_LIBRARY()

LICENSE(
    Apache-2.0 AND
    BSD-3-Clause
)

VERSION(v0.63.0)

SRCS(
    filters.go
    header.go
)

GO_TEST_SRCS(filters_test.go)

END()

RECURSE(
    gotest
)
