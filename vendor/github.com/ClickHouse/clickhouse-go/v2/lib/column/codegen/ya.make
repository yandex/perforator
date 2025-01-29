GO_PROGRAM()

LICENSE(Apache-2.0)

VERSION(v2.18.0)

SRCS(
    main.go
)

GO_EMBED_PATTERN(array.tpl)

GO_EMBED_PATTERN(column.tpl)

END()
