GO_LIBRARY()

LICENSE(BSD-3-Clause)

VERSION(v0.33.0)

SRCS(
    list.go
    table.go
)

GO_EMBED_PATTERN(data/children)

GO_EMBED_PATTERN(data/nodes)

GO_EMBED_PATTERN(data/text)

END()
