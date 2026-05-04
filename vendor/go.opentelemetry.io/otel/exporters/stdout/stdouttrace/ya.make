GO_LIBRARY()

LICENSE(
    Apache-2.0 AND
    BSD-3-Clause
)

VERSION(v1.38.0)

GO_SKIP_TESTS(TestExporterExportSpan)

SRCS(
    config.go
    doc.go
    trace.go
)

GO_XTEST_SRCS(
    example_test.go
    trace_test.go
)

END()

RECURSE(
    gotest
    internal
)
