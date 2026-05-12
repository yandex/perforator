GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.33.0)

SRCS(
    conditions.go
    doc.go
    errors.go
    firsthit_restmapper.go
    help.go
    interfaces.go
    lazy.go
    meta.go
    multirestmapper.go
    priority.go
    restmapper.go
)

GO_TEST_SRCS(
    conditions_test.go
    errors_test.go
    help_test.go
    meta_test.go
    multirestmapper_test.go
    priority_test.go
    restmapper_test.go
)

END()

RECURSE(
    gotest
    table
    testrestmapper
)
