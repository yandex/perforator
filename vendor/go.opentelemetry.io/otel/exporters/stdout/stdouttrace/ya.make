GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.34.0)

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
)
