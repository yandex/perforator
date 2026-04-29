GO_LIBRARY()

LICENSE(
    Apache-2.0 AND
    BSD-3-Clause
)

VERSION(v1.42.0)

SRCS(
    xxhash.go
)

GO_TEST_SRCS(xxhash_test.go)

END()

RECURSE(
    gotest
)
