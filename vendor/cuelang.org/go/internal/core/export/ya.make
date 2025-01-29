GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    adt.go
    bounds.go
    export.go
    expr.go
    extract.go
    label.go
    toposort.go
    value.go
)

GO_TEST_SRCS(toposort_test.go)

GO_XTEST_SRCS(
    # export_test.go
    # extract_test.go
    # value_test.go
)

END()

RECURSE(
    gotest
)
