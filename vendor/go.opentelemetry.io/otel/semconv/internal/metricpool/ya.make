GO_LIBRARY()

LICENSE(
    Apache-2.0 AND
    BSD-3-Clause
)

VERSION(v1.46.0)

ALL_GO_SRCS()

GO_TEST_SRCS(pool_test.go)

END()

RECURSE(
    gotest
)
