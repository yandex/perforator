GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.37.0)

SRCS(
    attr.go
    doc.go
    id.go
    number.go
    resource.go
    scope.go
    span.go
    status.go
    traces.go
    value.go
)

GO_TEST_SRCS(
    attr_test.go
    bench_test.go
    conv_test.go
    resource_test.go
    scope_test.go
    span_test.go
    traces_test.go
)

END()

RECURSE(
    gotest
)
