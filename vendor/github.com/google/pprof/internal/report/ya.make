GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.0.0-20250317173921-a4b03ec1a45e)

SRCS(
    package.go
    report.go
    shortnames.go
    source.go
    source_html.go
    stacks.go
    synth.go
)

GO_TEST_SRCS(
    package_test.go
    report_test.go
    shortnames_test.go
    source_test.go
    stacks_test.go
    synth_test.go
)

END()

RECURSE(
    # gotest
)
