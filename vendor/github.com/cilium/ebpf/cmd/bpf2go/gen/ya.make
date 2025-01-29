GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.17.1)

SRCS(
    compile.go
    doc.go
    output.go
    target.go
    types.go
)

GO_EMBED_PATTERN(output.tpl)

END()
