GO_LIBRARY()

LICENSE(
    Apache-2.0 AND
    BSD-3-Clause
)

VERSION(v1.40.0)

SRCS(
    instrumentation.go
)

GO_XTEST_SRCS(instrumentation_test.go)

END()

RECURSE(
    gotest
)
