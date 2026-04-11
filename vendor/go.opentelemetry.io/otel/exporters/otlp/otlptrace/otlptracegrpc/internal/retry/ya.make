GO_LIBRARY()

LICENSE(
    Apache-2.0 AND
    BSD-3-Clause
)

VERSION(v1.38.0)

SRCS(
    retry.go
)

GO_TEST_SRCS(retry_test.go)

END()

RECURSE(
    gotest
)
