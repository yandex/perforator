GO_LIBRARY()

LICENSE(
    Apache-2.0 AND
    BSD-3-Clause
)

VERSION(v1.40.0)

SRCS(
    assertion.go
    comparisons.go
)

GO_TEST_SRCS(assertion_test.go)

GO_XTEST_SRCS(example_test.go)

END()

RECURSE(
    gotest
)
