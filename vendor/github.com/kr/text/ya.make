GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.2.0)

SRCS(
    doc.go
    indent.go
    wrap.go
)

GO_TEST_SRCS(
    indent_test.go
    wrap_test.go
)

END()

RECURSE(
    cmd
    colwriter
    gotest
    mc
)
