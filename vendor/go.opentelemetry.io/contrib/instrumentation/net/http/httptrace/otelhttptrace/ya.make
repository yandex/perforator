GO_LIBRARY()

LICENSE(
    Apache-2.0 AND
    BSD-3-Clause
)

VERSION(v0.63.0)

SRCS(
    api.go
    clienttrace.go
    httptrace.go
    version.go
)

GO_TEST_SRCS(clienttrace_test.go)

GO_XTEST_SRCS(
    clienttracetest_test.go
    httptrace_test.go
    version_test.go
)

END()

RECURSE(
    gotest
    internal
)
