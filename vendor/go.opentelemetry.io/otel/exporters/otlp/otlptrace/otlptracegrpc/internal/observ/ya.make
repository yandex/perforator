GO_LIBRARY()

LICENSE(
    Apache-2.0 AND
    BSD-3-Clause
)

VERSION(v1.42.0)

SRCS(
    doc.go
    instrumentation.go
    target.go
)

GO_TEST_SRCS(target_test.go)

GO_XTEST_SRCS(instrumentation_test.go)

END()

RECURSE(
    gotest
)
