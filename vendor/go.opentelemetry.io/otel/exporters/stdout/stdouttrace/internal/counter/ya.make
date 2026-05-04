GO_LIBRARY()

LICENSE(
    Apache-2.0 AND
    BSD-3-Clause
)

VERSION(v1.38.0)

SRCS(
    counter.go
)

GO_TEST_SRCS(counter_test.go)

END()

RECURSE(
    gotest
)
