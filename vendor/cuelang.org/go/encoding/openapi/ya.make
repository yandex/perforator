GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    build.go
    crd.go
    cycle.go
    decode.go
    doc.go
    errors.go
    openapi.go
    orderedmap.go
    types.go
)

GO_XTEST_SRCS(
    decode_test.go
    openapi_test.go
)

END()

RECURSE(
    gotest
)
