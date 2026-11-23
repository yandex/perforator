GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.0.0-20260721132016-d427ff9ee9ad)

SRCS(
    arshal.go
    arshal_any.go
    arshal_default.go
    arshal_funcs.go
    arshal_inlined.go
    arshal_methods.go
    arshal_time.go
    doc.go
    errors.go
    fields.go
    fold.go
    intern.go
    options.go
)

END()

RECURSE(
    internal
    jsontext
)
