GO_LIBRARY()

LICENSE(MIT)

VERSION(v2.23.4)

SRCS(
    core_dsl.go
    decorator_dsl.go
    deprecated_dsl.go
    ginkgo_t_dsl.go
    reporting_dsl.go
    table_dsl.go
)

END()

RECURSE(
    config
    dsl
    extensions
    formatter
    ginkgo
    integration
    internal
    reporters
    types
)
