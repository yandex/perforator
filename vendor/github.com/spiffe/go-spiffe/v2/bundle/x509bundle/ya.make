GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v2.5.0)

SRCS(
    bundle.go
    doc.go
    set.go
    source.go
)

GO_XTEST_SRCS(
    bundle_test.go
    set_test.go
)

END()

RECURSE(
    gotest
)
