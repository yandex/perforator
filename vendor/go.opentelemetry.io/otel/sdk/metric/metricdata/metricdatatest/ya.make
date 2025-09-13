GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.36.0)

SRCS(
    assertion.go
    comparisons.go
)

GO_TEST_SRCS(assertion_test.go)

END()

RECURSE(
    gotest
)
