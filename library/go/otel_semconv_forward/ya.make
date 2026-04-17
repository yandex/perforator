GO_LIBRARY()

SRCS(
    doc.go
    semconv_compat.go
    semconv_gen.go
)

END()

RECURSE(
    internal
)
