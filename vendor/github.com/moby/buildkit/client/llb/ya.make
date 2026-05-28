GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.20.2)

SRCS(
    async.go
    definition.go
    diff.go
    exec.go
    fileop.go
    marshal.go
    merge.go
    meta.go
    resolver.go
    source.go
    sourcemap.go
    state.go
)

GO_TEST_SRCS(
    async_test.go
    definition_test.go
    exec_test.go
    fileop_test.go
    merge_test.go
    meta_test.go
    resolver_test.go
    state_test.go
)

END()

RECURSE(
    gotest
    sourceresolver
)
