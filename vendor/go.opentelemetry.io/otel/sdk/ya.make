GO_LIBRARY()

LICENSE(
    Apache-2.0 AND
    BSD-3-Clause
)

VERSION(v1.40.0)

SRCS(
    version.go
)

GO_XTEST_SRCS(version_test.go)

END()

RECURSE(
    gotest
    instrumentation
    internal
    resource
    trace
)
