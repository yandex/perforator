GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.34.3)

SRCS(
    cacheable_object.go
    conversion.go
    doc.go
    types.go
    validation.go
    zz_generated.deepcopy.go
)

END()

RECURSE(
    v1
)
