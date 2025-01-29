GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    attrs.go
    internal.go
)

GO_TEST_SRCS(attrs_test.go)

END()

RECURSE(
    astinternal
    ci
    cli
    cmd
    copy
    core
    cuetest
    cuetxtar
    diff
    encoding
    filetypes
    gotest
    source
    str
    task
    third_party
    types
    value
)
