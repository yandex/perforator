GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    build.go
    errors.go
    go.go
    imports.go
    index.go
    resolve.go
    runtime.go
)

GO_TEST_SRCS(resolve_test.go)

END()

RECURSE(
    gotest
)
