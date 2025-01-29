GO_LIBRARY()

LICENSE(
    Apache-2.0 AND
    BSD-3-Clause AND
    MIT
)

VERSION(v1.17.9)

SRCS(
    stdlib.go
)

GO_TEST_SRCS(stdlib_test.go)

END()

RECURSE(
    gotest
)
