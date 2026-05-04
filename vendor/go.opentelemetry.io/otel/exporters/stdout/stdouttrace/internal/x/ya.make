GO_LIBRARY()

LICENSE(
    Apache-2.0 AND
    BSD-3-Clause
)

VERSION(v1.38.0)

SRCS(
    x.go
)

GO_TEST_SRCS(x_test.go)

END()

RECURSE(
    gotest
)
